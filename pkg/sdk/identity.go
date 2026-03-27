package sdk

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"fmt"
	"os"
)

// DefaultIdentity implements the Identity interface using Ed25519.
type DefaultIdentity struct {
	publicKey  ed25519.PublicKey
	privateKey ed25519.PrivateKey
}

func NewIdentity() Identity {
	return &DefaultIdentity{}
}

// GenerateKeypair creates a new Ed25519 keypair for sovereign authentication.
func (id *DefaultIdentity) GenerateKeypair() (public, private []byte, err error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("identity: failed to generate Ed25519 keypair: %w", err)
	}
	id.publicKey = pub
	id.privateKey = priv
	return []byte(pub), []byte(priv), nil
}

// LoadKeypair reads an Ed25519 private key from a PEM file.
func (id *DefaultIdentity) LoadKeypair(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("identity: failed to read key file: %w", err)
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return fmt.Errorf("identity: invalid PEM data in %s", path)
	}

	if len(block.Bytes) != ed25519.PrivateKeySize {
		return fmt.Errorf("identity: invalid key size: got %d, want %d", len(block.Bytes), ed25519.PrivateKeySize)
	}

	id.privateKey = ed25519.PrivateKey(block.Bytes)
	id.publicKey = id.privateKey.Public().(ed25519.PublicKey)
	return nil
}

// SaveKeypair persists the private key as PEM to disk.
func (id *DefaultIdentity) SaveKeypair(path string) error {
	if id.privateKey == nil {
		return fmt.Errorf("identity: no keypair loaded")
	}
	block := &pem.Block{
		Type:  "CROM ED25519 PRIVATE KEY",
		Bytes: []byte(id.privateKey),
	}
	return os.WriteFile(path, pem.EncodeToMemory(block), 0600)
}

// Sign signs data with the loaded private key.
func (id *DefaultIdentity) Sign(data []byte) ([]byte, error) {
	if id.privateKey == nil {
		return nil, fmt.Errorf("identity: no private key loaded")
	}
	return ed25519.Sign(id.privateKey, data), nil
}

// Verify checks a signature against a public key.
func (id *DefaultIdentity) Verify(public, data, signature []byte) bool {
	if len(public) != ed25519.PublicKeySize {
		return false
	}
	return ed25519.Verify(ed25519.PublicKey(public), data, signature)
}

// PublicKey returns the loaded public key bytes.
func (id *DefaultIdentity) PublicKey() []byte {
	return []byte(id.publicKey)
}
