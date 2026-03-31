package entropy

import (
	"io"
	"math"
)

// Analyze reads up to sampleSize bytes from r and returns the Shannon Entropy (H).
// H varies from 0 (all same bytes) to 8 (complete randomness/encryption/compression).
func Analyze(r io.Reader, sampleSize int) (float64, []byte, error) {
	buf := make([]byte, sampleSize)
	n, err := io.ReadFull(r, buf)
	if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {
		return 0, nil, err
	}
	
	if n == 0 {
		return 0, nil, nil
	}

	freq := make(map[byte]int)
	for i := 0; i < n; i++ {
		freq[buf[i]]++
	}

	entropy := 0.0
	length := float64(n)
	for _, count := range freq {
		p := float64(count) / length
		entropy -= p * math.Log2(p)
	}

	return entropy, buf[:n], nil
}

// DetectHeuristicBypass checks magic bytes and entropy to decide if it's not compressible.
// It returns a boolean indicating if it should bypass Codebook/Delta processing instantly.
func DetectHeuristicBypass(entropy float64, buf []byte) bool {
	// Magic bytes checks for heavily compressed files
	if len(buf) > 4 {
		// PNG
		if buf[0] == 0x89 && buf[1] == 0x50 && buf[2] == 0x4E && buf[3] == 0x47 {
			return true
		}
		// WEBP (RIFF...WEBP)
		if string(buf[0:4]) == "RIFF" && len(buf) >= 12 && string(buf[8:12]) == "WEBP" {
			return true
		}
		// ZIP / JAR
		if buf[0] == 0x50 && buf[1] == 0x4B && buf[2] == 0x03 && buf[3] == 0x04 {
			return true
		}
		// GZIP
		if buf[0] == 0x1F && buf[1] == 0x8B {
			return true
		}
		// ELF Binaries
		if buf[0] == 0x7F && buf[1] == 0x45 && buf[2] == 0x4C && buf[3] == 0x46 {
			return true
		}
		// Se for GGUF, nós ainda queremos bypass da Lz4/Zstd, MAS precisamos flagar a camada
		if IsNeuralGGUF(buf) {
			return true
		}
		// JPEG/JPG
		if buf[0] == 0xFF && buf[1] == 0xD8 && buf[2] == 0xFF {
			return true
		}
		// GIF
		if string(buf[0:4]) == "GIF8" {
			return true
		}
	}

	// Shannon entropy limit
	// Highly unpredictable data like MP4, JPG yield > 7.7
	if entropy > 7.8 {
		return true
	}

	return false
}

// IsLowEntropy checks if data is extremely repetitive or highly compressible (e.g. all-zeros).
func IsLowEntropy(entropy float64) bool {
	return entropy < 1.0
}

// Shannon quickly calculates the entropy of an in-memory byte slice.
func Shannon(data []byte) float64 {
	if len(data) == 0 {
		return 0.0
	}
	
	freq := make(map[byte]int)
	for _, b := range data {
		freq[b]++
	}

	entropy := 0.0
	length := float64(len(data))
	for _, count := range freq {
		p := float64(count) / length
		entropy -= p * math.Log2(p)
	}

	return entropy
}

// IsNeuralGGUF detecta de forma contundente se os bytes percentem a um Payload Neural GGUF.
// Se positivo, a Engine V24 injetará esse arquivo via VFS Paging O(1), cortando os limites 
// nos tensores matriciais invés de offsets aleatórios.
func IsNeuralGGUF(buf []byte) bool {
	if len(buf) >= 4 && string(buf[0:4]) == "GGUF" {
		return true
	}
	return false
}

