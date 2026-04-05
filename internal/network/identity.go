//go:build !wasm

package network

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
)

// Identity struct stores the keys locally
type Identity struct {
	PrivKeyBytes []byte `json:"privKey"`
	PubKeyBytes  []byte `json:"pubKey"`
	PeerID       string `json:"peerID"`
}

func getIdentityPath() string {
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, ".crompressor", "keys")
	os.MkdirAll(dir, 0700)
	return filepath.Join(dir, "identity.json")
}

func getTrustPath() string {
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, ".crompressor", "keys")
	os.MkdirAll(dir, 0700)
	return filepath.Join(dir, "trust.json")
}

// GenerateIdentity creates a new Ed25519 keypair for libp2p
func GenerateIdentity() error {
	priv, pub, err := crypto.GenerateEd25519Key(rand.Reader)
	if err != nil {
		return err
	}

	pid, err := peer.IDFromPublicKey(pub)
	if err != nil {
		return err
	}

	privBytes, err := crypto.MarshalPrivateKey(priv)
	if err != nil {
		return err
	}

	pubBytes, err := crypto.MarshalPublicKey(pub)
	if err != nil {
		return err
	}

	ident := Identity{
		PrivKeyBytes: privBytes,
		PubKeyBytes:  pubBytes,
		PeerID:       pid.String(),
	}

	data, err := json.MarshalIndent(ident, "", "  ")
	if err != nil {
		return err
	}

	path := getIdentityPath()
	if err := os.WriteFile(path, data, 0600); err != nil {
		return err
	}
	
	fmt.Printf("Identidade P2P gerada: %s\nSalvo em: %s\n", pid.String(), path)
	return nil
}

// LoadIdentity loads the libp2p private key from disk
func LoadIdentity() (crypto.PrivKey, error) {
	path := getIdentityPath()
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("identidade nao encontrada, rode 'crompressor keys --gen'")
	}

	var ident Identity
	if err := json.Unmarshal(data, &ident); err != nil {
		return nil, err
	}

	return crypto.UnmarshalPrivateKey(ident.PrivKeyBytes)
}

// TrustPeer adds a peer ID to the Web of Trust
func TrustPeer(peerIDStr string) error {
	_, err := peer.Decode(peerIDStr)
	if err != nil {
		return fmt.Errorf("peer ID invalido: %w", err)
	}

	path := getTrustPath()
	var trusted []string

	data, err := os.ReadFile(path)
	if err == nil {
		json.Unmarshal(data, &trusted)
	}

	for _, p := range trusted {
		if p == peerIDStr {
			return nil // J  confiado
		}
	}

	trusted = append(trusted, peerIDStr)
	data, _ = json.MarshalIndent(trusted, "", "  ")
	return os.WriteFile(path, data, 0644)
}

// IsPeerTrusted checks if a peer is in the Web of Trust
func IsPeerTrusted(peerIDStr string) bool {
	path := getTrustPath()
	var trusted []string
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	json.Unmarshal(data, &trusted)
	for _, p := range trusted {
		if p == peerIDStr {
			return true
		}
	}
	return false
}
