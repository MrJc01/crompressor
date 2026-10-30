package semantic

import (
	"bytes"
	"testing"
)

func TestDetectBinaries(t *testing.T) {
	cases := map[string][]byte{
		"ELF":     {0x7f, 'E', 'L', 'F'},
		"ZIP":     {'P', 'K', 3, 4},
		"PNG":     {0x89, 'P', 'N', 'G'},
		"UNKNOWN": {},
	}
	for want, sample := range cases {
		if got := DetectHeuristicExtension(sample); got != want {
			t.Errorf("%v: %s, want %s", sample, got, want)
		}
	}
}

// Whatever the type, chunks must cover the input exactly and respect the limit.
func TestSplitCoversInput(t *testing.T) {
	inputs := map[string][]byte{
		"LINES":   bytes.Repeat([]byte("uma linha de log\n"), 500),
		"JSON":    bytes.Repeat([]byte(`{"a":1,"b":[1,2]},`), 500),
		"UNKNOWN": bytes.Repeat([]byte{1, 2, 3, 4}, 2000),
	}
	const maxSize = 1024
	for kind, data := range inputs {
		var joined []byte
		for _, c := range NewContextualChunker(kind, maxSize).Split(data) {
			if len(c.Data) > maxSize {
				t.Errorf("%s: chunk of %d bytes", kind, len(c.Data))
			}
			if uint64(len(joined)) != c.Offset {
				t.Errorf("%s: offset %d, want %d", kind, c.Offset, len(joined))
			}
			joined = append(joined, c.Data...)
		}
		if !bytes.Equal(joined, data) {
			t.Errorf("%s: chunks do not rebuild the input", kind)
		}
	}
}
