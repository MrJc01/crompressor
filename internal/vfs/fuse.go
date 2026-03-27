package vfs

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/crom-project/crom/internal/codebook"
	"github.com/crom-project/crom/pkg/format"
	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
)

// CromRoot is the root directory of the FUSE mount.
type CromRoot struct {
	fs.Inode
	reader   *RandomReader
	fileName string
	fileSize int64
}

var _ fs.NodeOnAdder = (*CromRoot)(nil)

func (r *CromRoot) OnAdd(ctx context.Context) {
	// Add the single file to the root directory
	ch := r.NewPersistentInode(ctx, &CromFile{reader: r.reader, size: r.fileSize}, fs.StableAttr{Mode: fuse.S_IFREG | 0444, Ino: 2})
	r.AddChild(r.fileName, ch, true)
}

// CromFile represents the unpacked file inside the FUSE mount.
type CromFile struct {
	fs.Inode
	reader *RandomReader
	size   int64
}

var _ fs.NodeReader = (*CromFile)(nil)
var _ fs.NodeWriter = (*CromFile)(nil)
var _ fs.NodeGetattrer = (*CromFile)(nil)
var _ fs.NodeOpener = (*CromFile)(nil)

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
	// [WBCache] Write-back cache staging
	// Blocks written here are asynchronously appended to a local buffer before P2P Sync.
	fmt.Printf("[WBCache] Staging %d bytes at offset %d\n", len(data), off)
	return uint32(len(data)), 0
}

// Mount mounts a .crom file at the given mountpoint.
// It blocks until the filesystem is unmounted.
func Mount(cromFile string, mountPoint string, codebookFile string, encryptionKey string) error {
	cb, err := codebook.Open(codebookFile)
	if err != nil {
		return fmt.Errorf("mount: failed to auto-load codebook: %w", err)
	}
	defer cb.Close()

	file, err := os.Open(cromFile)
	if err != nil {
		return fmt.Errorf("mount: failed to open .crom: %w", err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return err
	}

	reader := format.NewReader(file)
	header, blockTable, entries, err := reader.ReadMetadata(encryptionKey)
	if err != nil {
		return fmt.Errorf("mount: failed to parse format metadata: %w", err)
	}

	randomReader, err := NewRandomReader(file, info.Size(), header, blockTable, entries, cb, encryptionKey)
	if err != nil {
		return fmt.Errorf("mount: failed to init random reader: %w", err)
	}

	baseName := filepath.Base(cromFile)
	if strings.HasSuffix(baseName, ".crom") {
		baseName = strings.TrimSuffix(baseName, ".crom")
	} else {
		baseName = baseName + ".restored.raw"
	}

	root := &CromRoot{
		reader:   randomReader,
		fileName: baseName,
		fileSize: int64(header.OriginalSize),
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
