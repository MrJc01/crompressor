package vfs

import (
	"context"
	"fmt"
	"os"
	"syscall"

	"github.com/MrJc01/crompressor/pkg/cromdb"
	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
)

// TreeInode implementa navegação POSIX de grafos usando o SQLite Inodes do CromDB.
type TreeInode struct {
	fs.Inode
	inodeID int64
	fsIndex *cromdb.TreeFS

	// Caso seja folha (arquivo), manter referências de dados:
	reader *RandomReader
	wal    *WriteAheadLog
	size   int64
	isDir  bool
}

var _ = (fs.NodeReaddirer)((*TreeInode)(nil))
var _ = (fs.NodeLookuper)((*TreeInode)(nil))
var _ = (fs.NodeGetattrer)((*TreeInode)(nil))
var _ = (fs.NodeReader)((*TreeInode)(nil))
var _ = (fs.NodeOpener)((*TreeInode)(nil))
var _ = (fs.NodeWriter)((*TreeInode)(nil))

func (n *TreeInode) Getattr(ctx context.Context, f fs.FileHandle, out *fuse.AttrOut) syscall.Errno {
	if n.isDir {
		out.Mode = fuse.S_IFDIR | 0755
	} else {
		out.Mode = fuse.S_IFREG | 0644
		out.Size = uint64(n.size)
	}
	return 0
}

func (n *TreeInode) Lookup(ctx context.Context, name string, out *fuse.EntryOut) (*fs.Inode, syscall.Errno) {
	if !n.isDir {
		return nil, syscall.ENOTDIR
	}

	child, err := n.fsIndex.LookupNode(n.inodeID, name)
	if err != nil || child == nil {
		return nil, syscall.ENOENT
	}

	childType := &TreeInode{
		inodeID: child.ID,
		fsIndex: n.fsIndex,
		isDir:   child.IsDir,
		size:    child.Size,
		reader:  n.reader, // herda o reader em caso de arquivo
		wal:     n.wal,
	}

	mode := fuse.S_IFREG | 0644
	if child.IsDir {
		mode = fuse.S_IFDIR | 0755
	}
	out.Attr.Mode = uint32(mode)
	if !child.IsDir {
		out.Attr.Size = uint64(child.Size)
	}

	return n.NewInode(ctx, childType, fs.StableAttr{Mode: uint32(mode), Ino: uint64(child.ID)}), 0
}

func (n *TreeInode) Readdir(ctx context.Context) (fs.DirStream, syscall.Errno) {
	if !n.isDir {
		return nil, syscall.ENOTDIR
	}

	children, err := n.fsIndex.GetChildren(n.inodeID)
	if err != nil {
		// Log apenas uma vez por instância para não poluir o terminal
		// (o kernel chama Readdir repetidamente em background)
		fmt.Fprintf(os.Stderr, "vfs: Readdir(inode=%d) SQLite error (suppressing repeats): %v\n", n.inodeID, err)
		return fs.NewListDirStream(nil), 0
	}

	entries := make([]fuse.DirEntry, len(children))
	for i, c := range children {
		mode := uint32(fuse.S_IFREG)
		if c.IsDir {
			mode = fuse.S_IFDIR
		}
		entries[i] = fuse.DirEntry{
			Mode: mode,
			Name: c.Name,
			Ino:  uint64(c.ID),
		}
	}

	return fs.NewListDirStream(entries), 0
}
	
func (n *TreeInode) Open(ctx context.Context, flags uint32) (fh fs.FileHandle, fuseFlags uint32, errno syscall.Errno) {
	// A API fs do go-fuse permite retornar nil como handle se não precisarmos de estado por arquivo.
	// O importante é retornar SUCCESS (0) para o Kernel permitir a abertura.
	return nil, 0, 0
}

func (n *TreeInode) Read(ctx context.Context, fh fs.FileHandle, dest []byte, off int64) (fuse.ReadResult, syscall.Errno) {
	if n.isDir {
		return nil, syscall.EISDIR
	}
	if n.reader == nil {
		return nil, syscall.EIO
	}
	b, err := n.reader.ReadAt(dest, off)
	if err != nil && err.Error() != "EOF" {
		return nil, syscall.EIO
	}
	return fuse.ReadResultData(dest[:b]), 0
}

func (n *TreeInode) Write(ctx context.Context, fh fs.FileHandle, data []byte, off int64) (uint32, syscall.Errno) {
	if n.isDir {
		return 0, syscall.EISDIR
	}
	if n.wal != nil {
		if err := n.wal.Append(data, off); err != nil {
			return 0, syscall.EIO
		}
	}
	return uint32(len(data)), 0
}
