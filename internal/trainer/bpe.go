package trainer

import (
	"log"
)

// BPEBuilder implements a Byte-Pair Encoding (BPE) engine.
// It extracts highly repetitive sub-word patterns (semantic tokens)
// from raw text or binary data up to a maximum length limit.
type BPEBuilder struct {
	vocab      map[uint32][]byte // TokenID -> Raw Bytes
	maxLen     int               // Maximum allowed length for a merged token
	maxTokens  int               // Target vocabulary size (e.g. 8192)
}

// NewBPEBuilder initializes a new NLP Tokenizer builder.
func NewBPEBuilder(maxTokens, maxLen int) *BPEBuilder {
	vocab := make(map[uint32][]byte, maxTokens)
	// Base vocabulary: The 256 physical bytes
	for i := 0; i < 256; i++ {
		vocab[uint32(i)] = []byte{byte(i)}
	}

	return &BPEBuilder{
		vocab:     vocab,
		maxLen:    maxLen,
		maxTokens: maxTokens,
	}
}

// Train processes an entire dataset in memory to extract semantic tokens.
func (b *BPEBuilder) Train(data []byte) map[uint32][]byte {
	if len(data) == 0 {
		return b.vocab
	}

	// 1. Convert raw bytes to abstract Token space
	tokens := make([]uint32, len(data))
	for i, b := range data {
		tokens[i] = uint32(b)
	}

	type Pair struct {
		A, B uint32
	}

	// 2. Iteratively merge the most frequent pairs
	// We start assigning new tokens at ID 256.
	for nextTokenID := uint32(256); nextTokenID < uint32(b.maxTokens); {
		if len(tokens) < 2 {
			break
		}

		// a. Count pair frequencies
		counts := make(map[Pair]int)
		for i := 0; i < len(tokens)-1; i++ {
			p := Pair{tokens[i], tokens[i+1]}
			counts[p]++
		}

		// b. Find the most frequent pair that obeys length boundaries
		var bestPair Pair
		bestCount := -1

		for p, count := range counts {
			lenA := len(b.vocab[p.A])
			lenB := len(b.vocab[p.B])
			if lenA+lenB > b.maxLen {
				continue // Skip: the merged token would be too long for our LSH 128-byte limit!
			}

			if count > bestCount {
				bestCount = count
				bestPair = p
			}
		}

		// If no pairs can be merged anymore, or frequencies drop to 1, we stop.
		if bestCount < 2 {
			log.Printf("BPE: Stopping early at token %d, no repetitive pairs left.", nextTokenID)
			break
		}

		// c. Create the new Semantic Super-Token
		newTokenBytes := make([]byte, 0, len(b.vocab[bestPair.A])+len(b.vocab[bestPair.B]))
		newTokenBytes = append(newTokenBytes, b.vocab[bestPair.A]...)
		newTokenBytes = append(newTokenBytes, b.vocab[bestPair.B]...)
		b.vocab[nextTokenID] = newTokenBytes

		// d. Replace sequence in the integer stream inline
		newTokens := make([]uint32, 0, len(tokens))
		for i := 0; i < len(tokens); i++ {
			if i < len(tokens)-1 && tokens[i] == bestPair.A && tokens[i+1] == bestPair.B {
				newTokens = append(newTokens, nextTokenID)
				i++ // Skip the merged B token
			} else {
				newTokens = append(newTokens, tokens[i])
			}
		}

		tokens = newTokens // Swap buffer
		
		if nextTokenID%500 == 0 || nextTokenID == uint32(b.maxTokens-1) {
			log.Printf("BPE: Extracted Token %d [Freq: %d] [Bytes: %q]", nextTokenID, bestCount, b.vocab[nextTokenID])
		}

		nextTokenID++
	}

	return b.vocab
}
