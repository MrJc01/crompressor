//go:build !wasm

package network

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
)

const (
	// AuthProtocol is the protocol ID for the sovereignty handshake.
	AuthProtocol = "/crom/auth/2.0"
	
	// AuthTimeout is the maximum time allowed for the handshake.
	AuthTimeout = 10 * time.Second

	authNonceSize = 32
)

var errBrainMismatch = errors.New("Codebook diferente (Sovereignty violation)")

// setupSovereigntyAuth configures the host to require proof of holding the same
// brain before any file is served.
func (n *CromNode) setupSovereigntyAuth() {
	n.Host.SetStreamHandler(AuthProtocol, n.authHandler)
}

// authProof proves knowledge of the brain without revealing its hash. It is bound to
// the sender's identity and the other side's nonce, so it cannot be replayed or reflected.
func (n *CromNode) authProof(sender peer.ID, nonce []byte) []byte {
	mac := hmac.New(sha256.New, n.CodebookHash[:])
	mac.Write([]byte("crom-auth-v2"))
	mac.Write([]byte(sender))
	mac.Write(nonce)
	return mac.Sum(nil)
}

// handshake runs the same mutual challenge on both ends of the stream.
// The initiator waits for a final ack, so it only proceeds once the other side accepted it.
func (n *CromNode) handshake(s network.Stream, initiator bool) error {
	s.SetDeadline(time.Now().Add(AuthTimeout))

	ourNonce := make([]byte, authNonceSize)
	if _, err := rand.Read(ourNonce); err != nil {
		return err
	}
	if _, err := s.Write(ourNonce); err != nil {
		return fmt.Errorf("send nonce: %w", err)
	}
	remoteNonce := make([]byte, authNonceSize)
	if _, err := io.ReadFull(s, remoteNonce); err != nil {
		return fmt.Errorf("read nonce: %w", err)
	}
	if _, err := s.Write(n.authProof(n.Host.ID(), remoteNonce)); err != nil {
		return fmt.Errorf("send proof: %w", err)
	}
	remoteProof := make([]byte, sha256.Size)
	if _, err := io.ReadFull(s, remoteProof); err != nil {
		return fmt.Errorf("read proof: %w", err)
	}
	remote := s.Conn().RemotePeer()
	if !hmac.Equal(remoteProof, n.authProof(remote, ourNonce)) {
		return errBrainMismatch
	}
	n.authenticated.Store(remote, true)
	ack := make([]byte, 1)
	if initiator {
		if _, err := io.ReadFull(s, ack); err != nil {
			return fmt.Errorf("read ack: %w", err)
		}
		return nil
	}
	if _, err := s.Write(ack); err != nil {
		return fmt.Errorf("send ack: %w", err)
	}
	return nil
}

// authHandler processes incoming handshake streams.
func (n *CromNode) authHandler(s network.Stream) {
	defer s.Close()
	remote := s.Conn().RemotePeer()
	if err := n.handshake(s, false); err != nil {
		fmt.Printf("[Auth] ❌ Peer %s rejeitado: %v\n", remote, err)
		if errors.Is(err, errBrainMismatch) {
			s.Conn().Close()
		}
		return
	}
	fmt.Printf("[Auth] ✔ Peer %s autenticado no mesmo Codebook.\n", remote)
}

// AuthenticatePeer initiates the handshake with a remote peer.
// Must be called after establishing a connection before any other protocol.
func (n *CromNode) AuthenticatePeer(ctx context.Context, pid peer.ID) error {
	s, err := n.Host.NewStream(ctx, pid, AuthProtocol)
	if err != nil {
		return fmt.Errorf("auth: failed to open stream: %w", err)
	}
	defer s.Close()

	if err := n.handshake(s, true); err != nil {
		if errors.Is(err, errBrainMismatch) {
			n.Host.Network().ClosePeer(pid)
		}
		return fmt.Errorf("auth: %w", err)
	}
	return nil
}

// IsAuthenticated reports whether the peer proved it holds the same brain.
func (n *CromNode) IsAuthenticated(pid peer.ID) bool {
	_, ok := n.authenticated.Load(pid)
	return ok
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
