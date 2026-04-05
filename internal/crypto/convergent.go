package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
)

// ConvergentEncrypt implements Zero-Knowledge Convergent Encryption.
// The AES-GCM 256-bit key is derived by hashing the Plaintext along with a GlobalSecret.
// This guarantees that identical plaintexts ALWAYS generate the exact same Ciphertext
// (including the Nonce), enabling global P2P/DHT deduplication across different files or users.
func ConvergentEncrypt(globalSecret []byte, plaintext []byte) ([]byte, error) {
	// 1. Hash the Plaintext
	chunkHash := sha256.Sum256(plaintext)

	// 2. Derive the 32-byte AES Key via HMAC(Secret, ChunkHash)
	mac := hmac.New(sha256.New, globalSecret)
	mac.Write(chunkHash[:])
	aesKey := mac.Sum(nil)

	// 3. Prepare AES-GCM
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, fmt.Errorf("convergent: create cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("convergent: create gcm: %w", err)
	}

	// 4. Deterministic Nonce (first N bytes of the chunk's hash)
	// Because the ChunkHash is unique to the plaintext and we use a unique key per plaintext,
	// nonce reuse across the same Key is mathematically impossible unless it's the exact same plaintext.
	nonce := chunkHash[:aead.NonceSize()]

	// 5. Seal (Result: [Nonce][Ciphertext][Tag])
	// We prepend the nonce so Decrypt can read it directly.
	ciphertext := aead.Seal(nil, nonce, plaintext, nil)
	return append(nonce, ciphertext...), nil
}

// ConvergentDecrypt reverses the ConvergentEncrypt process.
// It requires the EXACT same globalSecret used during encryption.
func ConvergentDecrypt(globalSecret []byte, payload []byte) ([]byte, error) {
	// We don't have the plaintext anymore to derive the key,
	// BUT wait: convergent encryption means the key is derived from the plaintext.
	// We CANNOT decrypt it unless we either stream the derived key alongside it,
	// or we store the ChunkHash in the metadata!
	//
	// Correction for pure Convergent Encryption:
	// The overarching system MUST pass the original ChunkHash (which is usually the DHT Key or Merkle Leaf)
	// to this function because we cannot hash the plaintext we don't have yet.
	return nil, fmt.Errorf("convergent: use ConvergentDecryptWithHash instead")
}

// ConvergentDecryptWithHash decrypts the payload requiring the original Plaintext SHA-256 Hash.
// The hash is usually known by the Storage Engine as the BlockID or DHT Key.
func ConvergentDecryptWithHash(globalSecret []byte, originalPlaintextHash [32]byte, payload []byte) ([]byte, error) {
	// 1. Re-Derive the 32-byte AES Key via HMAC(Secret, ChunkHash)
	mac := hmac.New(sha256.New, globalSecret)
	mac.Write(originalPlaintextHash[:])
	aesKey := mac.Sum(nil)

	// 2. Prepare AES-GCM
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, fmt.Errorf("convergent: create cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("convergent: create gcm: %w", err)
	}

	nonceSize := aead.NonceSize()
	if len(payload) < nonceSize {
		return nil, fmt.Errorf("convergent: payload too short")
	}

	// 3. Extract Nonce and Ciphertext
	nonce := payload[:nonceSize]
	ciphertext := payload[nonceSize:]

	// 4. Decrypt
	plaintext, err := aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("convergent: decryption failed (wrong secret or corrupted data): %w", err)
	}

	// 5. Verify Hash Integrity to prevent Hash-Collision substitution attacks
	decryptedHash := sha256.Sum256(plaintext)
	if decryptedHash != originalPlaintextHash {
		return nil, fmt.Errorf("convergent: fatal integrity error, decrypted plaintext hash does not match block ID")
	}

	return plaintext, nil
}
