package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type CromMetrics struct {
	BytesSavedTotal        prometheus.Counter
	PackOpsTotal           prometheus.Counter
	UnpackOpsTotal         prometheus.Counter
	PackDuration           prometheus.Histogram
	CorruptBlocksRecovered prometheus.Counter
}

// Global instance to allow simplified registry logic
var GlobalMetrics *CromMetrics

func InitCromMetrics() {
	if GlobalMetrics != nil {
		return
	}

	GlobalMetrics = &CromMetrics{
		BytesSavedTotal: promauto.NewCounter(prometheus.CounterOpts{
			Name: "crom_bytes_saved_total",
			Help: "Total number of bytes saved by Crompressor across all pack operations",
		}),
		PackOpsTotal: promauto.NewCounter(prometheus.CounterOpts{
			Name: "crom_pack_operations_total",
			Help: "Total number of pack operations executed",
		}),
		UnpackOpsTotal: promauto.NewCounter(prometheus.CounterOpts{
			Name: "crom_unpack_operations_total",
			Help: "Total number of unpack operations executed",
		}),
		PackDuration: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "crom_pack_duration_seconds",
			Help:    "Duration of pack operations in seconds",
			Buckets: []float64{0.1, 0.5, 1.0, 5.0, 10.0, 30.0, 60.0},
		}),
		CorruptBlocksRecovered: promauto.NewCounter(prometheus.CounterOpts{
			Name: "crom_corrupt_blocks_recovered_total",
			Help: "Total number of corrupted frames ignored and zero-filled via tolerant unpack mode",
		}),
	}
}

// RecordPack updates the metrics after a pack operation.
func RecordPack(originalSize, packedSize uint64, duration time.Duration) {
	if GlobalMetrics == nil {
		return
	}
	GlobalMetrics.PackOpsTotal.Inc()
	GlobalMetrics.PackDuration.Observe(duration.Seconds())
	if originalSize > packedSize {
		GlobalMetrics.BytesSavedTotal.Add(float64(originalSize - packedSize))
	}
}

// RecordUnpack updates the metrics after an unpack operation.
func RecordUnpack(corruptBlocksRecovered int) {
	if GlobalMetrics == nil {
		return
	}
	GlobalMetrics.UnpackOpsTotal.Inc()
	if corruptBlocksRecovered > 0 {
		GlobalMetrics.CorruptBlocksRecovered.Add(float64(corruptBlocksRecovered))
	}
}
