package vfs

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/MrJc01/crompressor/internal/codebook"
	"github.com/MrJc01/crompressor/internal/remote"
	"github.com/MrJc01/crompressor/pkg/cromdb"
	"github.com/MrJc01/crompressor/pkg/format"
	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
)

type CromRoot struct {
	fs.Inode
	reader   *RandomReader
	fileName string
	fileSize int64
	wal      *WriteAheadLog
}

var _ fs.NodeOnAdder = (*CromRoot)(nil)

func (r *CromRoot) OnAdd(ctx context.Context) {
	// Add the single file to the root directory
	ch := r.NewPersistentInode(ctx, &CromFile{reader: r.reader, size: r.fileSize, wal: r.wal}, fs.StableAttr{Mode: fuse.S_IFREG | 0644, Ino: 2})
	r.AddChild(r.fileName, ch, true)
}

// CromFile represents the unpacked file inside the FUSE mount.
type CromFile struct {
	fs.Inode
	reader *RandomReader
	size   int64
	wal    *WriteAheadLog
}

var _ fs.NodeReader = (*CromFile)(nil)
var _ fs.NodeWriter = (*CromFile)(nil)
var _ fs.NodeGetattrer = (*CromFile)(nil)
var _ fs.NodeOpener = (*CromFile)(nil)
var _ fs.NodeFlusher = (*CromFile)(nil)

func (f *CromFile) Open(ctx context.Context, flags uint32) (fs.FileHandle, uint32, syscall.Errno) {
	return nil, 0, 0
}

func (f *CromFile) Getattr(ctx context.Context, fh fs.FileHandle, out *fuse.AttrOut) syscall.Errno {
	out.Mode = fuse.S_IFREG | 0644
	out.Size = uint64(f.size)
	return 0
}

func (f *CromFile) Read(ctx context.Context, fh fs.FileHandle, dest []byte, off int64) (fuse.ReadResult, syscall.Errno) {
	n, err := f.reader.ReadAt(dest, off)
	if err != nil && err.Error() != "EOF" {
		fmt.Fprintf(os.Stderr, "vfs: read error at off=%d len=%d: %v\n", off, len(dest), err)
		return nil, syscall.EIO
	}
	return fuse.ReadResultData(dest[:n]), 0
}

func (f *CromFile) Write(ctx context.Context, fh fs.FileHandle, data []byte, off int64) (uint32, syscall.Errno) {
	if f.wal != nil {
		err := f.wal.Append(data, off)
		if err != nil {
			return 0, syscall.EIO
		}
	} else {
		fmt.Printf("[WBCache] Staging %d bytes at offset %d (WAL Not Initialized)\n", len(data), off)
	}
	return uint32(len(data)), 0
}

func (f *CromFile) Flush(ctx context.Context, fh fs.FileHandle) syscall.Errno {
	if f.wal != nil {
		f.wal.forceFlush() // Commits directly to disk on close
	}
	return 0
}

// Mount mounts a .crom file at the given mountpoint.
// It blocks until the filesystem is unmounted.
func Mount(cromFile string, mountPoint string, codebookFile string, encryptionKey string, maxMB int) error {
	var cb *codebook.Reader
	var err error

	if strings.HasPrefix(codebookFile, "bitswap://") || strings.HasPrefix(codebookFile, "ipfs://") {
		// V20: P2P Bitswap Codebook Loading (Sharding on demand)
		fmt.Printf("🌐 Conectando à DHT Kademlia para injetar páginas do Codebook: %s\n", codebookFile)
		// cb, err = network.NewBitswapCodebook(codebookFile) // Implementação futura de p2p mmap
	} else {
		cb, err = codebook.Open(codebookFile)
	}

	if err != nil {
		return fmt.Errorf("mount: failed to auto-load codebook: %w", err)
	}
	defer cb.Close()

	var file io.ReaderAt
	var fileSize int64
	var fileCloser io.Closer

	if strings.HasPrefix(cromFile, "http://") || strings.HasPrefix(cromFile, "https://") {
		cr, err := remote.NewCloudReader(cromFile)
		if err != nil {
			return fmt.Errorf("mount: failed to init cloud reader: %w", err)
		}
		file = cr
		fileSize = cr.Size()
		fileCloser = io.NopCloser(nil) // CloudReader handles its own transient connections
	} else {
		localFile, err := os.Open(cromFile)
		if err != nil {
			return fmt.Errorf("mount: failed to open .crom: %w", err)
		}
		info, err := localFile.Stat()
		if err != nil {
			localFile.Close()
			return err
		}
		file = localFile
		fileSize = info.Size()
		fileCloser = localFile
	}
	defer fileCloser.Close()

	// io.Reader is fulfilled by both os.File and CloudReader
	readerInterface, ok := file.(io.Reader)
	if !ok {
		return fmt.Errorf("mount: file interface does not implement io.Reader")
	}

	reader := format.NewReader(readerInterface)
	header, blockTable, entries, err := reader.ReadMetadata(encryptionKey)
	if err != nil {
		return fmt.Errorf("mount: failed to parse format metadata: %w", err)
	}

	randomReader, err := NewRandomReader(file, fileSize, header, blockTable, entries, cb, encryptionKey, maxMB)
	if err != nil {
		return fmt.Errorf("mount: failed to init random reader: %w", err)
	}

	// Initialize Write-Ahead Log for Living Files
	walEngine := NewWriteAheadLog(cromFile)
	defer walEngine.Close()

	baseName := filepath.Base(cromFile)
	if strings.HasSuffix(baseName, ".crom") {
		baseName = strings.TrimSuffix(baseName, ".crom")
	} else {
		baseName = baseName + ".restored.raw"
	}

	fsIndex, err := cromdb.NewTreeFS(":memory:")
	if err != nil {
		return fmt.Errorf("mount: failed to init TreeFS mapping: %v", err)
	}

	err = fsIndex.IngestFileHash(baseName, int64(header.OriginalSize), "", 0644)
	if err != nil {
		return fmt.Errorf("mount: failed to index file hash: %w", err)
	}

	root := &TreeInode{
		inodeID: 1, // ID do diretório Root na B-Tree
		fsIndex: fsIndex,
		isDir:   true,
		reader:  randomReader,
		wal:     walEngine,
	}

	server, err := fs.Mount(mountPoint, root, &fs.Options{
		MountOptions: fuse.MountOptions{
			AllowOther: false, // Fix: previne erro de fusermount sem /etc/fuse.conf grant
			Name:       "cromfs",
		},
	})
	if err != nil {
		return fmt.Errorf("mount: fuse mount failed: %w", err)
	}

	// Start Sovereignty Watcher — auto-unmounts on codebook removal, signal, or key invalidation.
	watcher := NewSovereigntyWatcher(server, codebookFile, mountPoint)
	watcher.Start()

	fmt.Printf("✔ CROM Virtual Filesystem montado com sucesso!\n")
	fmt.Printf("  Arquivo:  %s\n", cromFile)
	fmt.Printf("  Ponto:    %s\n", mountPoint)
	fmt.Printf("  Codebook: %s\n", codebookFile)
	fmt.Println("  Soberania: Watcher ativo (codebook + signals)")
	fmt.Println("Pressione Ctrl+C para desmontar...")

	server.Wait()
	return nil
}
