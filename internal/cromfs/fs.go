//go:build linux || darwin

package cromfs

import (
	"context"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"github.com/MrJc01/crompressor/pkg/cromlib"
)

// CromFS represents the root of our sovereign deduplication filesystem.
// It intercepts file writes, chunking and compressing them via ACAC and LSH.
type CromFS struct {
	MountPoint   string
	OutputPool   string // Where the actual .crom files are saved
	CodebookPath string // The L1 codebook
}

// Root returns the root Node of the filesystem.
func (f *CromFS) Root() (fs.Node, error) {
	return &Dir{
		FS:   f,
		Path: f.OutputPool,
	}, nil
}

// Dir implements fs.Node and fs.HandleReadDirAller for directories.
type Dir struct {
	FS   *CromFS
	Path string
}

func (d *Dir) Attr(ctx context.Context, a *fuse.Attr) error {
	a.Inode = 1
	a.Mode = os.ModeDir | 0755
	return nil
}

func (d *Dir) Lookup(ctx context.Context, name string) (fs.Node, error) {
	// Everything is a virtual file that accepts writes
	return &File{
		FS:   d.FS,
		Name: name,
	}, nil
}

func (d *Dir) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	var ent []fuse.Dirent
	// Return empty dir for now (write-only interception concept)
	return ent, nil
}

// Create handles the creation of a new file interceptor.
func (d *Dir) Create(ctx context.Context, req *fuse.CreateRequest, resp *fuse.CreateResponse) (fs.Node, fs.Handle, error) {
	f := &File{
		FS:   d.FS,
		Name: req.Name,
	}
	f.initStream()
	return f, f, nil
}

// File implements fs.Node, fs.HandleWriter
type File struct {
	FS   *CromFS
	Name string

	mu     sync.Mutex
	pw     *io.PipeWriter
	waitCh chan error
}

func (f *File) Attr(ctx context.Context, a *fuse.Attr) error {
	a.Mode = 0644
	return nil
}

func (f *File) initStream() {
	pr, pw := io.Pipe()
	f.pw = pw
	f.waitCh = make(chan error, 1)

	// Spin a goroutine that consumes the pipe using PackStream!
	go func() {
		outPath := filepath.Join(f.FS.OutputPool, f.Name+".crom")
		
		// The sovereign storage interception
		opts := cromlib.PackOptions{
			UseACAC:       true,
			ACACDelimiter: '\n',
			ChunkSize:     128,
			Concurrency:   2,
			// For ZK Sovereignty, we can enable ConvergentEncryption here if a secret is provided
		}

		log.Printf("CromFS: streaming write to %s", outPath)
		metrics, err := cromlib.PackStream(pr, outPath, f.FS.CodebookPath, opts)
		if err != nil {
			log.Printf("CromFS: error packing %s: %v", outPath, err)
			f.waitCh <- err
			return
		}
		
		log.Printf("CromFS: finished %s [Saved %d chunks, %d bytes -> %d bytes]", outPath, metrics.TotalChunks, metrics.OriginalSize, metrics.PackedSize)
		f.waitCh <- nil
	}()
}

// Write intercepts the standard Posix write syscall and pumps it to the compressor.
func (f *File) Write(ctx context.Context, req *fuse.WriteRequest, resp *fuse.WriteResponse) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.pw == nil {
		f.initStream()
	}

	n, err := f.pw.Write(req.Data)
	resp.Size = n
	return err
}

// Flush signals the end of the file.
func (f *File) Flush(ctx context.Context, req *fuse.FlushRequest) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	
	if f.pw != nil {
		f.pw.Close()
		<-f.waitCh // Wait for compression to finish
		f.pw = nil
	}
	return nil
}

// Mount attaches the CromFS FUSE daemon to the mountpoint.
func Mount(mountPoint string, outputPool string, codebook string) error {
	c, err := fuse.Mount(mountPoint, fuse.FSName("cromfs"), fuse.Subtype("cromfs"))
	if err != nil {
		return err
	}
	defer c.Close()

	log.Printf("CromFS interceptor mounted at %s (output: %s)", mountPoint, outputPool)

	filesys := &CromFS{
		MountPoint:   mountPoint,
		OutputPool:   outputPool,
		CodebookPath: codebook,
	}

	if err := fs.Serve(c, filesys); err != nil {
		return err
	}

	return nil
}
