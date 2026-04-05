//go:build wasm

package cromdb

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Inode represents a file or folder in virtual structure
type Inode struct {
	ID        int64
	ParentID  int64
	Name      string
	IsDir     bool
	Size      int64
	DataHash  string
	Mode      uint32
	ModTime   time.Time
}

// TreeFS uses an in-memory mapped tree as a fallback for WASM / Edge
// (Provides generic fallback memory-map simulating IndexDB capabilities).
type TreeFS struct {
	mu     sync.RWMutex
	inodes map[int64]Inode
	nextID int64
}

// NewTreeFS initializes the memory-map metadata tree
func NewTreeFS(dsn string) (*TreeFS, error) {
	fs := &TreeFS{
		inodes: make(map[int64]Inode),
		nextID: 2,
	}
	
	// Root Node
	fs.inodes[1] = Inode{
		ID:       1,
		ParentID: 0,
		Name:     "",
		IsDir:    true,
		Mode:     16877, // S_IFDIR | 0755
		ModTime:  time.Now(),
	}
	
	return fs, nil
}

func (fs *TreeFS) getOrCreateDir(parentID int64, name string) (int64, error) {
	for id, n := range fs.inodes {
		if n.ParentID == parentID && n.Name == name && n.IsDir {
			return id, nil
		}
	}
	
	id := fs.nextID
	fs.nextID++
	
	fs.inodes[id] = Inode{
		ID:       id,
		ParentID: parentID,
		Name:     name,
		IsDir:    true,
		Mode:     16877,
		ModTime:  time.Now(),
	}
	return id, nil
}

// IngestFileHash populates the tree hierarchy up to the file
func (fs *TreeFS) IngestFileHash(fullPath string, size int64, fileHash string, mode uint32) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	parts := strings.Split(filepath.Clean(fullPath), string(filepath.Separator))
	if len(parts) > 0 && parts[0] == "" {
		parts = parts[1:]
	}

	parentID := int64(1)
	
	for i := 0; i < len(parts)-1; i++ {
		dirName := parts[i]
		newParent, err := fs.getOrCreateDir(parentID, dirName)
		if err != nil {
			return err
		}
		parentID = newParent
	}

	fileName := parts[len(parts)-1]
	
	// Replace if exists
	var existingID int64 = 0
	for id, n := range fs.inodes {
		if n.ParentID == parentID && n.Name == fileName && !n.IsDir {
			existingID = id
			break
		}
	}
	
	if existingID > 0 {
		n := fs.inodes[existingID]
		n.Size = size
		n.DataHash = fileHash
		n.Mode = mode
		fs.inodes[existingID] = n
	} else {
		id := fs.nextID
		fs.nextID++
		fs.inodes[id] = Inode{
			ID:       id,
			ParentID: parentID,
			Name:     fileName,
			IsDir:    false,
			Size:     size,
			DataHash: fileHash,
			Mode:     mode,
			ModTime:  time.Now(),
		}
	}
	
	return nil
}

// GetChildren returns children nodes of a given parent
func (fs *TreeFS) GetChildren(parentID int64) ([]Inode, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	
	var res []Inode
	for _, n := range fs.inodes {
		if n.ParentID == parentID {
			res = append(res, n)
		}
	}
	return res, nil
}

// LookupNode finds a node by parentID and name
func (fs *TreeFS) LookupNode(parentID int64, name string) (*Inode, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	
	for _, n := range fs.inodes {
		if n.ParentID == parentID && n.Name == name {
			// Copy
			nCpy := n
			return &nCpy, nil
		}
	}
	
	return nil, nil // not found
}
