// Package vfs provides public access to CROM virtual filesystem operations.
//
// This package re-exports functions from internal/vfs for use by
// satellite repositories (crompressor-gui, etc).
package vfs

import (
	"github.com/MrJc01/crompressor/internal/vfs"
)

// Mount mounts a .crom file as a FUSE filesystem at the given mount point.
func Mount(cromFile string, mountPoint string, codebookFile string, encryptionKey string, maxMB int) error {
	return vfs.Mount(cromFile, mountPoint, codebookFile, encryptionKey, maxMB)
}
