// Package network provides public access to CROM P2P networking operations.
//
// This package re-exports types and functions from internal/network for use by
// satellite repositories (crompressor-gui, crompressor-sync, etc).
package network

import (
	"github.com/MrJc01/crompressor/internal/network"
)

// CromNode wraps the internal network.CromNode for public use.
type CromNode = network.CromNode

// SyncProtocol wraps the internal network.SyncProtocol for public use.
type SyncProtocol = network.SyncProtocol

// NewCromNode creates and starts a libp2p host bound to the given codebook.
func NewCromNode(codebookPath string, listenPort int, dataDir string, encKey string) (*CromNode, error) {
	return network.NewCromNode(codebookPath, listenPort, dataDir, encKey)
}

// NewSyncProtocol registers the sync stream handler.
func NewSyncProtocol(node *CromNode) *SyncProtocol {
	return network.NewSyncProtocol(node)
}
