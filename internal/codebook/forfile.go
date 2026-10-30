package codebook

import (
	"fmt"

	"github.com/MrJc01/crompressor/internal/crypto"
	"github.com/klauspost/compress/zstd"
)

// OpenForFile returns the brain embedded in a .crom file or, without one, the one at path.
func OpenForFile(microSize uint32, micro []byte, encrypted bool, salt []byte, path, encryptionKey string) (*Reader, error) {
	if microSize == 0 {
		if path == "" {
			return nil, fmt.Errorf("este arquivo precisa do cérebro usado no pack (--codebook)")
		}
		cb, err := Open(path)
		if err != nil {
			return nil, fmt.Errorf("failed to open codebook: %w", err)
		}
		return cb, nil
	}
	if encrypted {
		dec, err := crypto.Decrypt(crypto.DeriveKey([]byte(encryptionKey), salt), micro)
		if err != nil {
			return nil, fmt.Errorf("decrypt embedded brain: %w", err)
		}
		micro = dec
	}
	dec, err := zstd.NewReader(nil, zstd.WithDecoderConcurrency(1))
	if err != nil {
		return nil, err
	}
	defer dec.Close()
	brain, err := dec.DecodeAll(micro, nil)
	if err != nil {
		return nil, fmt.Errorf("decompress embedded brain: %w", err)
	}
	return OpenFromBytes(brain)
}
