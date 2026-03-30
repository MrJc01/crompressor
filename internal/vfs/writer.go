package vfs

import (
	"bytes"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/MrJc01/crompressor/pkg/cromlib"
)

// WriteAheadLog manages memory buffering for FUSE writes, preventing the
// .crom file from being hammered with appending patches byte-by-byte (which would ruin compression).
type WriteAheadLog struct {
	mu            sync.Mutex
	buffer        *bytes.Buffer
	cromFilePath  string
	lastWriteTime time.Time
	done          chan struct{}
}

// NewWriteAheadLog creates a new WAL that flushes automatically after quiet periods.
func NewWriteAheadLog(cromFilePath string) *WriteAheadLog {
	wal := &WriteAheadLog{
		buffer:       new(bytes.Buffer),
		cromFilePath: cromFilePath,
		done:         make(chan struct{}),
	}
	// Start an asynchronous flush worker
	go wal.flushWorker()
	return wal
}

// Append stages a write operation to memory.
func (wal *WriteAheadLog) Append(data []byte, offset int64) error {
	wal.mu.Lock()
	defer wal.mu.Unlock()

	// In a complete implementation, this would handle seeking to 'offset'
	// and patching a full memory-mapped mirror. For this prototype of
	// Append-only V9 LSM, we just append to the buffer simulating a unified diff log.
	wal.buffer.Write(data)
	wal.lastWriteTime = time.Now()

	return nil
}

// flushWorker runs in the background and applies mutations to the physical .crom file.
func (wal *WriteAheadLog) flushWorker() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-wal.done:
			return
		case <-ticker.C:
			wal.tryFlush()
		}
	}
}

// tryFlush checks if enough time has passed since the last write to safely commit to disk.
func (wal *WriteAheadLog) tryFlush() {
	wal.mu.Lock()
	if wal.buffer.Len() == 0 || time.Since(wal.lastWriteTime) < 1*time.Second {
		wal.mu.Unlock()
		return
	}

	// Capture payload and reset buffer
	payload := make([]byte, wal.buffer.Len())
	copy(payload, wal.buffer.Bytes())
	wal.buffer.Reset()
	wal.mu.Unlock()

	// Perform physical disk IO
	err := wal.commitToDisk(payload)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[VFS WAL] Error flushing mutation to disk: %v\n", err)
	} else {
		fmt.Printf("[VFS WAL] Flushed %d bytes of semantic delta to %s\n", len(payload), wal.cromFilePath)
	}
}

// forceFlush ignores the cooldown timer and flushes immediately.
func (wal *WriteAheadLog) forceFlush() {
	wal.mu.Lock()
	if wal.buffer.Len() == 0 {
		wal.mu.Unlock()
		return
	}

	payload := make([]byte, wal.buffer.Len())
	copy(payload, wal.buffer.Bytes())
	wal.buffer.Reset()
	wal.mu.Unlock()

	err := wal.commitToDisk(payload)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[VFS WAL] Error flushing mutation to disk: %v\n", err)
	} else {
		fmt.Printf("[VFS WAL] Flushed %d bytes of semantic delta to %s\n", len(payload), wal.cromFilePath)
	}
}

// commitToDisk opens the .crom file in append mode and calls the mutator engine.
func (wal *WriteAheadLog) commitToDisk(payload []byte) error {
	file, err := os.OpenFile(wal.cromFilePath, os.O_RDWR, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	// Apply O(1) LSM Append Mutation
	return cromlib.AppendMutation(file, payload)
}

// Close forces a final flush and stops the background worker.
func (wal *WriteAheadLog) Close() {
	close(wal.done)
	wal.forceFlush()
}
