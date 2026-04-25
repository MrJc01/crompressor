// Package crypto provides public access to CROM cryptographic operations.
//
// This package re-exports functions from internal/crypto for use by
// satellite repositories (crompressor-sync, crompressor-gui, etc).
package crypto

import (
	"github.com/MrJc01/crompressor/internal/crypto"
)

// DeriveKey derives an AES-256 key from a password and salt using Argon2.
func DeriveKey(password []byte, salt []byte) []byte {
	return crypto.DeriveKey(password, salt)
}

// GenerateSalt generates a cryptographically secure random salt.
func GenerateSalt() ([]byte, error) {
	return crypto.GenerateSalt()
}

// Encrypt encrypts plaintext with AES-256-GCM.
func Encrypt(key []byte, plaintext []byte) ([]byte, error) {
	return crypto.Encrypt(key, plaintext)
}

// Decrypt decrypts AES-256-GCM ciphertext.
func Decrypt(key []byte, ciphertext []byte) ([]byte, error) {
	return crypto.Decrypt(key, ciphertext)
}

// ConvergentEncrypt encrypts plaintext deterministically using content-derived key.
func ConvergentEncrypt(globalSecret []byte, plaintext []byte) ([]byte, error) {
	return crypto.ConvergentEncrypt(globalSecret, plaintext)
}

// ConvergentDecrypt decrypts convergent-encrypted payload.
func ConvergentDecrypt(globalSecret []byte, payload []byte) ([]byte, error) {
	return crypto.ConvergentDecrypt(globalSecret, payload)
}

// VerifyDilithium verifies a Dilithium post-quantum signature.
func VerifyDilithium(pubKey []byte, signature []byte, payload []byte) bool {
	return crypto.VerifyDilithium(pubKey, signature, payload)
}
