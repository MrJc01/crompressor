package metrics

import (
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestInitCromMetrics(t *testing.T) {
	InitCromMetrics()
	if GlobalMetrics == nil {
		t.Fatal("GlobalMetrics should not be nil after init")
	}

	// Double init shouldn't panic
	InitCromMetrics()
}

func TestRecordPack(t *testing.T) {
	registry := prometheus.NewRegistry()
	
	gm := &CromMetrics{
		BytesSavedTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "test_bytes",
		}),
		PackOpsTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "test_pack",
		}),
		UnpackOpsTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "test_unpack",
		}),
		PackDuration: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "test_duration",
			Buckets: []float64{0.1, 0.5, 1.0},
		}),
		CorruptBlocksRecovered: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "test_corrupt",
		}),
	}
	registry.MustRegister(gm.BytesSavedTotal, gm.PackOpsTotal, gm.PackDuration)

	// Save original and restore it after
	original := GlobalMetrics
	defer func() { GlobalMetrics = original }()
	GlobalMetrics = gm

	RecordPack(1000, 200, 2*time.Second)

	err := testutil.GatherAndCompare(registry, strings.NewReader(`
		# HELP test_bytes 
		# TYPE test_bytes counter
		test_bytes 800
		# HELP test_duration 
		# TYPE test_duration histogram
		test_duration_bucket{le="0.1"} 0
		test_duration_bucket{le="0.5"} 0
		test_duration_bucket{le="1"} 0
		test_duration_bucket{le="+Inf"} 1
		test_duration_sum 2
		test_duration_count 1
		# HELP test_pack 
		# TYPE test_pack counter
		test_pack 1
	`), "test_bytes", "test_pack", "test_duration")

	if err != nil {
		t.Fatalf("unexpected metrics output: %v", err)
	}
}

func TestRecordUnpack(t *testing.T) {
	registry := prometheus.NewRegistry()
	
	gm := &CromMetrics{
		UnpackOpsTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "test_unpack",
		}),
		CorruptBlocksRecovered: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "test_corrupt",
		}),
	}
	registry.MustRegister(gm.UnpackOpsTotal, gm.CorruptBlocksRecovered)

	original := GlobalMetrics
	defer func() { GlobalMetrics = original }()
	GlobalMetrics = gm

	RecordUnpack(5)

	err := testutil.GatherAndCompare(registry, strings.NewReader(`
		# HELP test_corrupt 
		# TYPE test_corrupt counter
		test_corrupt 5
		# HELP test_unpack 
		# TYPE test_unpack counter
		test_unpack 1
	`), "test_unpack", "test_corrupt")

	if err != nil {
		t.Fatalf("unexpected metrics unpack output: %v", err)
	}
}
