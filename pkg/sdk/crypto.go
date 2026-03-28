package sdk

import (
	"fmt"
	"os"

	"github.com/MrJc01/crompressor/internal/crypto"
)

// Crypto exposes AES-256-GCM encryption to the GUI and external consumers.
type Crypto interface {
	EncryptFile(inputPath, outputPath, passphrase string) error
	DecryptFile(inputPath, outputPath, passphrase string) error
	DeriveKey(passphrase string) (key []byte, salt []byte, err error)
}

// DefaultCrypto wraps internal/crypto for SDK consumers.
type DefaultCrypto struct{}

func NewCrypto() Crypto {
	return &DefaultCrypto{}
}

// EncryptFile reads a file, encrypts it with AES-256-GCM, and writes the output.
// Output format: [salt 16 bytes][encrypted data]
func (c *DefaultCrypto) EncryptFile(inputPath, outputPath, passphrase string) error {
	plaintext, err := os.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("crypto: read input: %w", err)
	}

	salt, err := crypto.GenerateSalt()
	if err != nil {
		return fmt.Errorf("crypto: generate salt: %w", err)
	}

	key := crypto.DeriveKey([]byte(passphrase), salt)
	ciphertext, err := crypto.Encrypt(key, plaintext)
	if err != nil {
		return fmt.Errorf("crypto: encrypt: %w", err)
	}

	// Prepend salt to output
	output := append(salt, ciphertext...)
	if err := os.WriteFile(outputPath, output, 0644); err != nil {
		return fmt.Errorf("crypto: write output: %w", err)
	}

	return nil
}

// DecryptFile reads an encrypted file (salt+ciphertext) and decrypts it.
func (c *DefaultCrypto) DecryptFile(inputPath, outputPath, passphrase string) error {
	data, err := os.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("crypto: read input: %w", err)
	}

	if len(data) < 16 {
		return fmt.Errorf("crypto: file too small to contain salt")
	}

	salt := data[:16]
	ciphertext := data[16:]

	key := crypto.DeriveKey([]byte(passphrase), salt)
	plaintext, err := crypto.Decrypt(key, ciphertext)
	if err != nil {
		return fmt.Errorf("crypto: decrypt: %w", err)
	}

	if err := os.WriteFile(outputPath, plaintext, 0644); err != nil {
		return fmt.Errorf("crypto: write output: %w", err)
	}

	return nil
}

// DeriveKey generates a 256-bit key from a passphrase using PBKDF2.
func (c *DefaultCrypto) DeriveKey(passphrase string) (key []byte, salt []byte, err error) {
	salt, err = crypto.GenerateSalt()
	if err != nil {
		return nil, nil, err
	}
	key = crypto.DeriveKey([]byte(passphrase), salt)
	return key, salt, nil
}
