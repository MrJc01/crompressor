package vfs

import (
	"fmt"
	"io"
	"sync"

	"github.com/MrJc01/crompressor/internal/chunker"
	"github.com/MrJc01/crompressor/internal/codebook"
	"github.com/MrJc01/crompressor/internal/crypto"
	"github.com/MrJc01/crompressor/internal/delta"
	"github.com/MrJc01/crompressor/pkg/format"
)

// RandomReader provides an io.ReaderAt interface over a .crom file.
type RandomReader struct {
	file         io.ReaderAt
	fileSize     int64
	header       *format.Header
	blockTable   []uint32
	blockOffsets []int64 // precalculated absolute offsets in the .crom file
	entries      []format.ChunkEntry
	cb           *codebook.Reader
	memCache     *MemoryCache
	derivedKey   []byte
	dataOffset   int64 // Absolute offset where raw passthrough data or the first block starts

	mu sync.Mutex // Protects cache/disk reads to avoid redundant decompression of the same block
}

// NewRandomReader opens a .crom file for random access.
// File must be kept open by the caller.
// We expect exactly the data from format.Reader.Read(), minus the compDeltaPool, but because
// we want stream reading of the pool, we compute offsets here.
func NewRandomReader(f io.ReaderAt, fileSize int64, header *format.Header, blockTable []uint32, entries []format.ChunkEntry, cb *codebook.Reader, encryptionKey string, maxMB int) (*RandomReader, error) {
	if header.Version < format.Version2 {
		return nil, fmt.Errorf("vfs: only Version 2+ formats support Random Access")
	}

	rr := &RandomReader{
		file:       f,
		fileSize:   fileSize,
		header:     header,
		blockTable: blockTable,
		entries:    entries,
		cb:         cb,
		memCache:   NewMemoryCache(maxMB),
	}

	if header.IsEncrypted {
		if encryptionKey == "" {
			return nil, fmt.Errorf("vfs: file is encrypted but no key was provided")
		}
		rr.derivedKey = crypto.DeriveKey([]byte(encryptionKey), header.Salt[:])
	}

	// Calculate absolute offsets for each block in the file
	// Block Table is immediately after Header
	// Then ChunkTable
	tableSize := int(header.ChunkCount) * int(format.GetEntrySize(header.Version))
	if header.IsEncrypted {
		tableSize += 28
	}

	hSize := format.HeaderSizeV2
	if header.Version == format.Version4 {
		hSize = format.HeaderSizeV4
	} else if header.Version == format.Version5 {
		hSize = format.HeaderSizeV5
	} else if header.Version == format.Version6 || header.Version == format.Version7 {
		hSize = format.HeaderSizeV6
	} else if header.Version >= format.Version8 {
		hSize = format.HeaderSizeV8 + int(header.MicroDictSize)
	}

	baseOffset := int64(hSize + len(blockTable)*4 + tableSize)

	rr.blockOffsets = make([]int64, len(blockTable))
	current := baseOffset
	for i, size := range blockTable {
		rr.blockOffsets[i] = current
		current += int64(size)
	}
	rr.dataOffset = baseOffset

	return rr, nil
}

// ReadAt satisfies io.ReaderAt, allowing FUSE to read specific byte ranges O(1).
func (rr *RandomReader) ReadAt(dest []byte, off int64) (int, error) {
	if off >= int64(rr.header.OriginalSize) {
		return 0, io.EOF
	}

	bytesToRead := int64(len(dest))
	if off+bytesToRead > int64(rr.header.OriginalSize) {
		bytesToRead = int64(rr.header.OriginalSize) - off
	}

	dest = dest[:bytesToRead]
	
	// Fast path for Passthrough files (0 chunks)
	if rr.header.ChunkCount == 0 {
		return rr.file.ReadAt(dest, rr.dataOffset+off)
	}

	bytesRead := 0

	for bytesRead < int(bytesToRead) {
		currentOff := off + int64(bytesRead)
		cSize := int64(rr.header.ChunkSize)
		if cSize == 0 {
			cSize = int64(chunker.DefaultChunkSize)
		}
		chunkIndex := currentOff / cSize
		chunkOffset := currentOff % cSize

		if chunkIndex >= int64(len(rr.entries)) {
			break
		}

		entry := rr.entries[chunkIndex]
		blockID := uint32(chunkIndex / format.ChunksPerBlock)

		// ===========================
		// 🚀 L2 CHUNK CACHE BYPASS
		// ===========================
		var reconstructedChunk []byte
		
		if cachedChunk, ok := rr.memCache.Get(int64(chunkIndex)); ok {
			reconstructedChunk = cachedChunk
		} else {
			// Get uncompressed Delta Pool for this block if not in L2
			pool, err := rr.loadBlockPool(blockID)
			if err != nil {
				return bytesRead, fmt.Errorf("vfs: read block %d: %w", blockID, err)
			}
	
			// Calculate localized block start offset (the global stream offset of the first chunk in this block)
			blockStartChunkIdx := int64(blockID) * int64(format.ChunksPerBlock)
			blockStartGlobalOffset := rr.entries[blockStartChunkIdx].DeltaOffset
	
			entryLocalOffset := entry.DeltaOffset - blockStartGlobalOffset
	
			endOffset := entryLocalOffset + uint64(entry.DeltaSize)
			if endOffset > uint64(len(pool)) {
				return bytesRead, fmt.Errorf("vfs: delta bounds error on chunk %d", chunkIndex)
			}
	
			res := pool[entryLocalOffset:endOffset]
	
			if entry.CodebookID == format.LiteralCodebookID {
				reconstructedChunk = res
			} else {
			isPatch := (entry.CodebookID & format.FlagIsPatch) != 0
			// Mask out Tier bits and Patch flag (clear upper 4 bits)
			cleanID := entry.CodebookID & 0x0FFFFFFFFFFFFFFF
			pattern, err := rr.cb.Lookup(cleanID)
			if err != nil {
				return bytesRead, fmt.Errorf("vfs: lookup codeword %d: %w", cleanID, err)
			}

			usablePattern := pattern
			if uint32(len(usablePattern)) > entry.OriginalSize {
				usablePattern = usablePattern[:entry.OriginalSize]
			}

			if isPatch {
				reconstructedChunk, err = delta.ApplyPatch(usablePattern, res)
				if err != nil {
					reconstructedChunk = res
				}
			} else {
				if uint32(len(res)) > entry.OriginalSize {
					res = res[:entry.OriginalSize]
				}
				reconstructedChunk = delta.Apply(usablePattern, res)
			}
		}
		} // Closes L2 Cache else

		// Clamp reconstructedChunk to entry.OriginalSize
		if uint32(len(reconstructedChunk)) > entry.OriginalSize {
			reconstructedChunk = reconstructedChunk[:entry.OriginalSize]
		}
		
		// L2 CACHE INJECTION
		if entry.CodebookID != format.LiteralCodebookID {
			cacheCopy := make([]byte, len(reconstructedChunk))
			copy(cacheCopy, reconstructedChunk)
			rr.memCache.Put(int64(chunkIndex), cacheCopy)
		}

		// How much of this chunk do we need to copy?
		chunkRemaining := int64(entry.OriginalSize) - chunkOffset
		needed := int64(len(dest)) - int64(bytesRead)
		toCopy := chunkRemaining
		if needed < toCopy {
			toCopy = needed
		}
		if chunkOffset+toCopy > int64(len(reconstructedChunk)) {
			toCopy = int64(len(reconstructedChunk)) - chunkOffset
			if toCopy < 0 {
				toCopy = 0
			}
		}

		copy(dest[bytesRead:bytesRead+int(toCopy)], reconstructedChunk[chunkOffset:chunkOffset+toCopy])
		bytesRead += int(toCopy)
	}

	if bytesRead == 0 {
		return 0, io.EOF
	}

	return bytesRead, nil
}

// loadBlockPool reads an encrypted Zstd frame from disk, or returns it from cache.
func (rr *RandomReader) loadBlockPool(blockID uint32) ([]byte, error) {
	if pool, ok := rr.memCache.Get(blockID); ok {
		return pool, nil
	}

	// Force single-thread the block extraction to prevent duplicate I/O and CPU spikes
	rr.mu.Lock()
	defer rr.mu.Unlock()

	// Check cache again inside lock
	if pool, ok := rr.memCache.Get(blockID); ok {
		return pool, nil
	}

	if blockID >= uint32(len(rr.blockOffsets)) {
		return nil, fmt.Errorf("invalid block ID %d", blockID)
	}

	fileOff := rr.blockOffsets[blockID]
	blockSize := rr.blockTable[blockID]

	buf := make([]byte, blockSize)
	if _, err := rr.file.ReadAt(buf, fileOff); err != nil && err != io.EOF {
		return nil, fmt.Errorf("read block frame: %w", err)
	}

	if rr.header.IsEncrypted {
		dec, err := crypto.Decrypt(rr.derivedKey, buf)
		if err != nil {
			return nil, fmt.Errorf("decrypt block frame: %w", err)
		}
		buf = dec
	}

	pool, err := delta.DecompressPool(buf)
	if err != nil {
		return nil, fmt.Errorf("decompress pool: %w", err)
	}

	rr.memCache.Put(blockID, pool)
	return pool, nil
}
