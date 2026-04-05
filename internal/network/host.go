//go:build !wasm

// Package network implements the CROM P2P networking layer using libp2p.
//
// The network is sovereign: only peers that share the same Codebook BuildHash
// can communicate. This is enforced at the protocol level via a handshake
// on /crom/auth/1.0.
package network

import (
	"context"
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"

	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/transport/tcp"
	libquic "github.com/libp2p/go-libp2p/p2p/transport/quic"

	"github.com/MrJc01/crompressor/internal/codebook"
)

// CromNode is the main P2P node for the CROM network.
type CromNode struct {
	Host         host.Host
	DHT          *dht.IpfsDHT
	PubSub       *pubsub.PubSub
	CodebookHash [32]byte // SHA-256 BuildHash — defines the network partition
	DataDir      string   // Directory containing local .crom files
	EncKey       string   // AES passphrase for encrypted files
	CodebookPath string   // Path to the .cromdb file

	ctx    context.Context
	cancel context.CancelFunc
}

// NewCromNode creates and starts a libp2p host bound to the given codebook.
// The node identity (Ed25519 keypair) is persisted in dataDir/peer.key.
func NewCromNode(codebookPath string, listenPort int, dataDir string, encKey string) (*CromNode, error) {
	ctx, cancel := context.WithCancel(context.Background())

	// 1. Load codebook to get BuildHash
	cb, err := codebook.Open(codebookPath)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("network: open codebook: %w", err)
	}
	buildHash := cb.BuildHash()
	cb.Close()

	// 2. Ensure data directory exists
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		cancel()
		return nil, fmt.Errorf("network: create data dir: %w", err)
	}

	// 3. Load or generate Ed25519 identity
	privKey, err := loadOrGenerateKey(filepath.Join(dataDir, "peer.key"))
	if err != nil {
		cancel()
		return nil, fmt.Errorf("network: identity: %w", err)
	}

	// 4. Create libp2p host
	listenAddr := fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", listenPort)
	listenAddrQUIC := fmt.Sprintf("/ip4/0.0.0.0/udp/%d/quic-v1", listenPort)

	h, err := libp2p.New(
		libp2p.Identity(privKey),
		libp2p.ListenAddrStrings(listenAddr, listenAddrQUIC),
		libp2p.Transport(tcp.NewTCPTransport),
		libp2p.Transport(libquic.NewTransport),
		libp2p.NATPortMap(),
		libp2p.EnableNATService(),
		libp2p.EnableHolePunching(),
	)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("network: create host: %w", err)
	}

	node := &CromNode{
		Host:         h,
		CodebookHash: buildHash,
		DataDir:      dataDir,
		EncKey:       encKey,
		CodebookPath: codebookPath,
		ctx:          ctx,
		cancel:       cancel,
	}

	// 5. Setup Protocol Handlers
	node.setupSovereigntyAuth()

	// 6. Start Discovery (mDNS)
	if err := node.setupDiscovery(); err != nil {
		cancel()
		return nil, fmt.Errorf("network: setup discovery: %w", err)
	}

	// 7. Start GossipSub (Announcements)
	if err := node.setupGossipSub(); err != nil {
		cancel()
		return nil, fmt.Errorf("network: setup gossip: %w", err)
	}

	return node, nil
}

// PeerID returns this node's peer ID as a string.
func (n *CromNode) PeerID() peer.ID {
	return n.Host.ID()
}

// Addrs returns the multiaddrs this node is listening on.
func (n *CromNode) Addrs() []string {
	addrs := n.Host.Addrs()
	result := make([]string, len(addrs))
	for i, a := range addrs {
		result[i] = fmt.Sprintf("%s/p2p/%s", a, n.Host.ID())
	}
	return result
}

// Stop gracefully shuts down the node.
func (n *CromNode) Stop() error {
	n.cancel()
	if n.DHT != nil {
		n.DHT.Close()
	}
	return n.Host.Close()
}

// Context returns the node's context.
func (n *CromNode) Context() context.Context {
	return n.ctx
}

// --- Identity Management ---

func loadOrGenerateKey(path string) (crypto.PrivKey, error) {
	// Try to load existing key
	if data, err := os.ReadFile(path); err == nil {
		priv, err := crypto.UnmarshalPrivateKey(data)
		if err != nil {
			return nil, fmt.Errorf("unmarshal key: %w", err)
		}
		return priv, nil
	}

	// Generate new Ed25519 key
	priv, _, err := crypto.GenerateEd25519Key(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate key: %w", err)
	}

	// Persist to disk
	raw, err := crypto.MarshalPrivateKey(priv)
	if err != nil {
		return nil, fmt.Errorf("marshal key: %w", err)
	}

	if err := os.WriteFile(path, raw, 0600); err != nil {
		return nil, fmt.Errorf("write key: %w", err)
	}

	return priv, nil
}
