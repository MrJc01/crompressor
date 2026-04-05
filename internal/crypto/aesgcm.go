package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"

	"golang.org/x/crypto/pbkdf2"
)

// DeriveKey implements PBKDF2 for password-based key derivation.
// Uses 100,000 iterations of SHA-256 for resistance to brute-force.
func DeriveKey(password []byte, salt []byte) []byte {
	return pbkdf2.Key(password, salt, 100000, 32, sha256.New)
}

// GenerateSalt creates a cryptographically secure random 16-byte salt.
func GenerateSalt() ([]byte, error) {
	salt := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, err
	}
	return salt, nil
}

// Encrypt payload with AES-256-GCM.
// Returns a byte slice containing: [nonce (12 bytes)][ciphertext + tag]
func Encrypt(key []byte, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	// Never use more than 2^32 random nonces with a given key because of the risk of a repeat.
	nonce := make([]byte, aesgcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	// seal appends the ciphertext and Mac tag directly to the nonce
	ciphertext := aesgcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// Decrypt an AES-256-GCM payload formatted by Encrypt.
func Decrypt(key []byte, ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := aesgcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("crypto: ciphertext too short")
	}

	nonce, encryptedData := ciphertext[:nonceSize], ciphertext[nonceSize:]

	plaintext, err := aesgcm.Open(nil, nonce, encryptedData, nil)
	if err != nil {
		return nil, fmt.Errorf("crypto: decryption failed (wrong key or corrupted data): %w", err)
	}

	return plaintext, nil
}
