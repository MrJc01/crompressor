package crypto

import (
	"bytes"
	"crypto/sha256"
	"testing"
)

func TestConvergentEncryption_Determinism(t *testing.T) {
	secret := []byte("CrompressorSovereignKey99")
	plaintext := []byte("My critical isolated JSON Log 200 OK")

	cipher1, err := ConvergentEncrypt(secret, plaintext)
	if err != nil {
		t.Fatalf("crypto: enc1 failed: %v", err)
	}

	// Encrypt exact same payload again
	cipher2, err := ConvergentEncrypt(secret, plaintext)
	if err != nil {
		t.Fatalf("crypto: enc2 failed: %v", err)
	}

	if !bytes.Equal(cipher1, cipher2) {
		t.Fatal("crypto: ConvergentEncrypt did not produce deterministic output for identical string")
	}

	hash := sha256.Sum256(plaintext)
	decrypted, err := ConvergentDecryptWithHash(secret, hash, cipher1)
	if err != nil {
		t.Fatalf("crypto: dec failed: %v", err)
	}

	if !bytes.Equal(plaintext, decrypted) {
		t.Fatalf("crypto: decrypted plaintext corrupted: expected %s, got %s", plaintext, decrypted)
	}
}

func TestConvergentEncryption_UniqueHashes(t *testing.T) {
	secret := []byte("Global")
	p1 := []byte("Chunk A")
	p2 := []byte("Chunk B")

	c1, _ := ConvergentEncrypt(secret, p1)
	c2, _ := ConvergentEncrypt(secret, p2)

	if bytes.Equal(c1, c2) {
		t.Fatal("crypto: distinct plaintexts resulted in identical ciphertext collision")
	}
}

func TestConvergentEncryption_WrongSecret(t *testing.T) {
	p1 := []byte("Chunk A")
	hash := sha256.Sum256(p1)

	c1, err := ConvergentEncrypt([]byte("Secret1"), p1)
	if err != nil {
		t.Fatal(err)
	}

	_, err = ConvergentDecryptWithHash([]byte("Secret2"), hash, c1)
	if err == nil {
		t.Fatal("crypto: Decrypt accepted wrong global secret")
	}
}
