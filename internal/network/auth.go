//go:build !wasm

package network

import (
	"bytes"
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
)

const (
	// AuthProtocol is the protocol ID for the sovereignty handshake.
	AuthProtocol = "/crom/auth/1.0"
	
	// AuthTimeout is the maximum time allowed for the handshake.
	AuthTimeout = 10 * time.Second
)

// setupSovereigntyAuth configures the host to require a Codebook BuildHash match
// for any incoming or outgoing connection to be considered trusted.
func (n *CromNode) setupSovereigntyAuth() {
	// Set the stream handler for incoming auth handshakes
	n.Host.SetStreamHandler(AuthProtocol, n.authHandler)
}

// authHandler processes incoming handshake streams.
// It receives the remote node's CodebookHash and sends its own.
// If they don't match, the connection is instantly closed.
func (n *CromNode) authHandler(s network.Stream) {
	defer s.Close()

	// Set deadline for the handshake
	s.SetDeadline(time.Now().Add(AuthTimeout))

	// 1. Send our CodebookHash
	if _, err := s.Write(n.CodebookHash[:]); err != nil {
		fmt.Printf("[Auth] Failed to send CodebookHash to %s: %v\n", s.Conn().RemotePeer(), err)
		return
	}

	// 2. Receive remote CodebookHash
	remoteHash := make([]byte, 32)
	if _, err := s.Read(remoteHash); err != nil {
		fmt.Printf("[Auth] Failed to read CodebookHash from %s: %v\n", s.Conn().RemotePeer(), err)
		return
	}

	// 3. Verify Sovereignty
	if !bytes.Equal(n.CodebookHash[:], remoteHash) {
		fmt.Printf("[Auth] ❌ SOBERANIA REJEITADA: Peer %s possui Codebook diferente.\n", s.Conn().RemotePeer())
		s.Conn().Close() // Terminate the connection entirely
		return
	}

	fmt.Printf("[Auth] ✔ Peer %s autenticado no mesmo Codebook.\n", s.Conn().RemotePeer())
}

// AuthenticatePeer initiates the handshake with a remote peer.
// Must be called after establishing a connection before any other protocol.
func (n *CromNode) AuthenticatePeer(ctx context.Context, pid peer.ID) error {
	s, err := n.Host.NewStream(ctx, pid, AuthProtocol)
	if err != nil {
		return fmt.Errorf("auth: failed to open stream: %w", err)
	}
	defer s.Close() // Will close write side, wait for response, then full close

	s.SetDeadline(time.Now().Add(AuthTimeout))

	// 1. Send our CodebookHash
	if _, err := s.Write(n.CodebookHash[:]); err != nil {
		return fmt.Errorf("auth: failed to send hash: %w", err)
	}

	// 2. Receive remote CodebookHash
	remoteHash := make([]byte, 32)
	if _, err := s.Read(remoteHash); err != nil {
		return fmt.Errorf("auth: failed to read hash: %w", err)
	}

	// 3. Verify Sovereignty
	if !bytes.Equal(n.CodebookHash[:], remoteHash) {
		n.Host.Network().ClosePeer(pid) // Disconnect
		return fmt.Errorf("auth: Codebook mismatch (Sovereignty violation)")
	}

	return nil
}

// discoveryNotifee implements mdns.Notifee interface
type discoveryNotifee struct {
	h    host.Host
	node *CromNode
	mu   sync.Mutex
}

// HandlePeerFound is called when mDNS discovers a peer in the local network
func (n *discoveryNotifee) HandlePeerFound(pi peer.AddrInfo) {
	n.mu.Lock()
	defer n.mu.Unlock()

	// Ignore self
	if pi.ID == n.h.ID() {
		return
	}

	fmt.Printf("[Descoberta] Peer mDNS encontrado: %s\n", pi.ID.String())

	// Background connection to avoid blocking mdns
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := n.h.Connect(ctx, pi); err != nil {
			fmt.Printf("[Descoberta] Falha ao conectar em %s: %v\n", pi.ID.String(), err)
			return
		}

		// Perform Sovereignty Handshake
		if err := n.node.AuthenticatePeer(ctx, pi.ID); err != nil {
			fmt.Printf("[Descoberta] Autenticação falhou com %s: %v\n", pi.ID.String(), err)
			return
		}

		fmt.Printf("[Descoberta] Conexão Soberana estabelecida com %s\n", pi.ID.String())
	}()
}

// setupDiscovery configures mDNS for local network peer discovery
func (n *CromNode) setupDiscovery() error {
	// The service tag includes the first 16 chars of the codebook hash
	// so we only discover peers likely on the same network partition.
	serviceTag := fmt.Sprintf("_crom_%x._tcp", n.CodebookHash[:8])

	ser := mdns.NewMdnsService(n.Host, serviceTag, &discoveryNotifee{h: n.Host, node: n})
	if err := ser.Start(); err != nil {
		return fmt.Errorf("discovery: failed to start mdns: %w", err)
	}

	return nil
}
