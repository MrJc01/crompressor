package entropy

import (
	"bytes"
	"math"
	"testing"
)

func TestShannon(t *testing.T) {
	if got := Shannon(nil); got != 0 {
		t.Errorf("empty: %v", got)
	}
	if got := Shannon(bytes.Repeat([]byte{7}, 100)); got != 0 {
		t.Errorf("constant: %v", got)
	}
	if got := Shannon([]byte{0, 1, 0, 1}); math.Abs(got-1) > 1e-9 {
		t.Errorf("two symbols: %v, want 1", got)
	}
	all := make([]byte, 256)
	for i := range all {
		all[i] = byte(i)
	}
	if got := Shannon(all); math.Abs(got-8) > 1e-9 {
		t.Errorf("uniform: %v, want 8", got)
	}
}

func TestAnalyzeMatchesShannonOnSample(t *testing.T) {
	data := []byte("abcabcabcabc")
	score, sample, err := Analyze(bytes.NewReader(data), 6)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(sample, data[:6]) {
		t.Errorf("sample: %q", sample)
	}
	if math.Abs(score-Shannon(data[:6])) > 1e-9 {
		t.Errorf("score: %v", score)
	}
}

func TestDetectHeuristicBypass(t *testing.T) {
	cases := []struct {
		name    string
		entropy float64
		buf     []byte
		want    bool
	}{
		{"png", 1, []byte{0x89, 'P', 'N', 'G', 0}, true},
		{"zip", 1, []byte{'P', 'K', 3, 4, 0}, true},
		{"gzip", 1, []byte{0x1f, 0x8b, 0, 0, 0}, true},
		{"jpeg", 1, []byte{0xff, 0xd8, 0xff, 0, 0}, true},
		{"high entropy", 7.9, []byte("texto"), true},
		{"text", 4.5, []byte("texto comum"), false},
	}
	for _, c := range cases {
		if got := DetectHeuristicBypass(c.entropy, c.buf); got != c.want {
			t.Errorf("%s: %v, want %v", c.name, got, c.want)
		}
	}
}
