package chunker

import (
	"bytes"
	"testing"
)

func TestSemanticChunker_ValidJSONLines(t *testing.T) {
	data := []byte(`{"log":"Starting App"}
{"log":"Connected"}
{"log":"Error DB"}
`)
	
	c := NewSemanticChunker('\n', 1024)
	chunks := c.Split(data)

	if len(chunks) != 3 {
		t.Fatalf("Expected 3 chunks, got %d", len(chunks))
	}

	for i, chunk := range chunks {
		if chunk.Data[len(chunk.Data)-1] != '\n' {
			t.Errorf("Chunk %d did not end with a newline", i)
		}
	}

	reassembled := Reassemble(chunks)
	if !bytes.Equal(data, reassembled) {
		t.Fatal("Reassembled JSON Lines did not match original.")
	}
}

func TestSemanticChunker_ExceedsMaxSize(t *testing.T) {
	// A JSON Line exactly 20 bytes long
	data := []byte("0123456789012345678\n")
	
	// Set max size to 10
	c := NewSemanticChunker('\n', 10)
	chunks := c.Split(data)

	// Since max size is 10, it should chop the 20 byte line into two chunks
	if len(chunks) != 2 {
		t.Fatalf("Expected exactly 2 chunks due to max size chop, got %d", len(chunks))
	}

	if len(chunks[0].Data) != 10 {
		t.Errorf("First chunk should be forced to length 10, got %d", len(chunks[0].Data))
	}

	reassembled := Reassemble(chunks)
	if !bytes.Equal(data, reassembled) {
		t.Fatal("Reassembled oversized JSON line did not match original.")
	}
}
