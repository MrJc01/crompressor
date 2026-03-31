package codebook

import (
	"context"
	"log"
	"sync"
	"time"
)

// DecayEngine manages the lifecycle of cached codebook chunks to prevent OOM
// on long-running nodes (Research 20: Codebook Radioactive Decay).
type DecayEngine struct {
	reader  *Reader
	heatmap map[uint64]int64 // chunkID -> LastTouchedTimestamp
	mu      sync.Mutex
	ctx     context.Context
	cancel  context.CancelFunc
}

// NewDecayEngine initializes the Least-Frequently-Used codebook garbage collector.
func NewDecayEngine(r *Reader) *DecayEngine {
	ctx, cancel := context.WithCancel(context.Background())
	return &DecayEngine{
		reader:  r,
		heatmap: make(map[uint64]int64),
		ctx:     ctx,
		cancel:  cancel,
	}
}

// Touch updates the last access time for a chunk. Called by the HNSW search.
func (d *DecayEngine) Touch(chunkID uint64) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.heatmap[chunkID] = time.Now().Unix()
}

// Start begins the radioactive decay background process.
func (d *DecayEngine) Start(decayWindow time.Duration, tickInterval time.Duration) {
	go func() {
		ticker := time.NewTicker(tickInterval)
		defer ticker.Stop()
		for {
			select {
			case <-d.ctx.Done():
				return
			case <-ticker.C:
				d.decay(decayWindow)
			}
		}
	}()
}

// Stop gracefully shuts down the garbage collector.
func (d *DecayEngine) Stop() {
	d.cancel()
}

// decay performs the logical eviction of cold codes.
func (d *DecayEngine) decay(decayWindow time.Duration) {
	d.mu.Lock()
	defer d.mu.Unlock()

	now := time.Now().Unix()
	var evicted int
	for id, ts := range d.heatmap {
		if now-ts > int64(decayWindow.Seconds()) {
			// Radioactive decay triggered: chunk is cold
			delete(d.heatmap, id)
			// Reader.lruCache is unexported but accessible in the same package
			if d.reader != nil && d.reader.lruCache != nil {
				delete(d.reader.lruCache, id)
				// SRE Concept: In mmap, unix.Madvise(MADV_DONTNEED) happens here
				evicted++
			}
		}
	}
	if evicted > 0 {
		log.Printf("☢️ [SRE] Codebook Decay: %d chunks expurgados da L1 cache\n", evicted)
	}
}
