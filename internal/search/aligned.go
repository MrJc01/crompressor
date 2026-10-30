package search

import (
	"github.com/cespare/xxhash/v2"

	"github.com/MrJc01/crompressor/internal/chunker"
	"github.com/MrJc01/crompressor/internal/codebook"
)

// AlignedMinCodewordSize is the codeword size from which a brain is used for
// deduplication: chunks are matched whole, wherever they appear in the input.
const AlignedMinCodewordSize = 1024

const rollingPrime uint64 = 1099511628211

// AlignedChunker cuts the input around the codewords of a brain, so an insertion
// only costs the bytes around it instead of misaligning everything that follows.
type AlignedChunker struct {
	cb      *codebook.Reader
	size    int
	topMult uint64 // rollingPrime^(size-1)
	windows map[uint64]uint64
}

// NewAlignedChunker indexes every codeword of cb by its rolling hash.
func NewAlignedChunker(cb *codebook.Reader) *AlignedChunker {
	size := int(cb.CodewordSize())
	ac := &AlignedChunker{
		cb:      cb,
		size:    size,
		topMult: 1,
		windows: make(map[uint64]uint64, cb.CodewordCount()),
	}
	for i := 1; i < size; i++ {
		ac.topMult *= rollingPrime
	}
	for id := uint64(0); id < cb.CodewordCount(); id++ {
		ac.windows[rollingHash(cb.LookupUnsafe(id))] = id
	}
	return ac
}

func rollingHash(data []byte) uint64 {
	var h uint64
	for _, b := range data {
		h = h*rollingPrime + uint64(b)
	}
	return h
}

// Split implements chunker.Chunker.
func (ac *AlignedChunker) Split(data []byte) []chunker.Chunk {
	var chunks []chunker.Chunk
	emit := func(from, to int) {
		chunks = append(chunks, chunker.Chunk{
			Data:   data[from:to],
			Offset: uint64(from),
			Size:   uint32(to - from),
			Hash:   xxhash.Sum64(data[from:to]),
		})
	}
	emitLiteral := func(from, to int) {
		for from < to {
			end := from + ac.size
			if end > to {
				end = to
			}
			emit(from, end)
			from = end
		}
	}

	literalStart, pos := 0, 0
	for pos+ac.size <= len(data) {
		h := rollingHash(data[pos : pos+ac.size])
		for {
			if id, ok := ac.windows[h]; ok && string(ac.cb.LookupUnsafe(id)) == string(data[pos:pos+ac.size]) {
				break
			}
			if pos+ac.size >= len(data) {
				emitLiteral(literalStart, len(data))
				return chunks
			}
			h = (h-uint64(data[pos])*ac.topMult)*rollingPrime + uint64(data[pos+ac.size])
			pos++
		}
		emitLiteral(literalStart, pos)
		emit(pos, pos+ac.size)
		pos += ac.size
		literalStart = pos
	}
	emitLiteral(literalStart, len(data))
	return chunks
}
