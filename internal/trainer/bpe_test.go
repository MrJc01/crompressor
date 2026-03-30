package trainer_test

import (
	"bytes"
	"testing"

	"github.com/MrJc01/crompressor/internal/trainer"
)

func TestBPEBuilder(t *testing.T) {
	bpe := trainer.NewBPEBuilder(300, 128)

	// Create a repetition of "ABABABA CDCD"
	// To trick the frequency algorithm into merging "AB", then "CD"
	var buf bytes.Buffer
	for i := 0; i < 1000; i++ {
		buf.WriteString("ABABABA CDCD ")
	}
	
	vocab := bpe.Train(buf.Bytes())

	// Assert that BPE successfully recognized repeating semantic blocks and added new Tokens
	if len(vocab) <= 256 {
		t.Fatalf("BPE failed to extract new tokens. Vocab size: %d", len(vocab))
	}

	foundLargeToken := false
	for id := uint32(256); id < uint32(len(vocab)); id++ {
		if len(vocab[id]) > 4 {
			foundLargeToken = true
			break
		}
	}

	if !foundLargeToken {
		t.Errorf("BPE did not extract any large semantic tokens from highly repeated text.")
	}
}
