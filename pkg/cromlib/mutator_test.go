package cromlib

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/MrJc01/crompressor/pkg/format"
)

func TestV9_AppendMutation(t *testing.T) {
	tmpDir := t.TempDir()
	cromFile := filepath.Join(tmpDir, "test.crom")

	// Create a dummy format.Version9 file with a minimal header
	f, err := os.Create(cromFile)
	if err != nil {
		t.Fatal(err)
	}
	f.Write([]byte("CROM"))                    // Magic (4 bytes)
	f.Write([]byte{byte(format.Version9), 0})  // Version9 (2 bytes)

	// Pad fake header to base size (141 - 6 = 135 bytes zero)
	pad := make([]byte, format.HeaderSizeV8-6)
	f.Write(pad)
	f.Close()

	headerOnlySize := format.HeaderSizeV8

	// 1. Open the file for appending
	file, err := os.OpenFile(cromFile, os.O_RDWR, 0644)
	if err != nil {
		t.Fatal(err)
	}

	// 2. Append a mutation
	payload := []byte("Hello World!")
	err = AppendMutation(file, payload)
	if err != nil {
		t.Fatalf("AppendMutation failed: %v", err)
	}
	file.Close()

	// 3. Read the file back and validate structure
	data, err := os.ReadFile(cromFile)
	if err != nil {
		t.Fatal(err)
	}

	expectedSize := headerOnlySize + format.V9MutationHeaderSize + len(payload)
	if len(data) != expectedSize {
		t.Fatalf("Expected file size %d, got %d", expectedSize, len(data))
	}

	// 4. Validate CMUT magic at tail
	mutStart := headerOnlySize
	magic := string(data[mutStart : mutStart+4])
	if magic != "CMUT" {
		t.Fatalf("Expected magic 'CMUT' at offset %d, got '%s'", mutStart, magic)
	}

	// 5. Parse the mutation header
	mutHeader, err := format.ParseV9MutationHeader(data[mutStart : mutStart+format.V9MutationHeaderSize])
	if err != nil {
		t.Fatalf("ParseV9MutationHeader failed: %v", err)
	}

	if mutHeader.DiffPatchSize != uint32(len(payload)) {
		t.Fatalf("Expected DiffPatchSize %d, got %d", len(payload), mutHeader.DiffPatchSize)
	}

	// 6. Validate payload bytes
	actualPayload := data[mutStart+format.V9MutationHeaderSize:]
	if !bytes.Equal(actualPayload, payload) {
		t.Fatalf("Payload mismatch: got %q, want %q", actualPayload, payload)
	}

	t.Logf("✔ V9 Mutation appended successfully: magic=CMUT, size=%d, payload=%q", mutHeader.DiffPatchSize, string(actualPayload))
}

func TestV9_MultipleAppendMutations(t *testing.T) {
	tmpDir := t.TempDir()
	cromFile := filepath.Join(tmpDir, "multi.crom")

	// Create minimal V9 file
	f, _ := os.Create(cromFile)
	f.Write([]byte("CROM"))
	f.Write([]byte{byte(format.Version9), 0})
	pad := make([]byte, format.HeaderSizeV8-6)
	f.Write(pad)
	f.Close()

	// Append 3 mutations sequentially
	payloads := []string{"First", "Second", "Third"}
	for _, p := range payloads {
		file, err := os.OpenFile(cromFile, os.O_RDWR, 0644)
		if err != nil {
			t.Fatal(err)
		}
		if err := AppendMutation(file, []byte(p)); err != nil {
			t.Fatalf("AppendMutation(%q) failed: %v", p, err)
		}
		file.Close()
	}

	// Read back and traverse all 3 mutation headers
	data, _ := os.ReadFile(cromFile)
	cursor := format.HeaderSizeV8

	for i, expected := range payloads {
		if cursor+format.V9MutationHeaderSize > len(data) {
			t.Fatalf("Mutation #%d: cursor %d exceeds file length %d", i, cursor, len(data))
		}

		mh, err := format.ParseV9MutationHeader(data[cursor : cursor+format.V9MutationHeaderSize])
		if err != nil {
			t.Fatalf("Mutation #%d parse failed: %v", i, err)
		}

		payloadStart := cursor + format.V9MutationHeaderSize
		payloadEnd := payloadStart + int(mh.DiffPatchSize)
		actual := string(data[payloadStart:payloadEnd])

		if actual != expected {
			t.Fatalf("Mutation #%d: got %q, want %q", i, actual, expected)
		}

		t.Logf("✔ Mutation #%d: ts=%d payload=%q", i, mh.Timestamp, actual)
		cursor = payloadEnd
	}

	if cursor != len(data) {
		t.Fatalf("Trailing garbage: cursor=%d, file_len=%d", cursor, len(data))
	}
}
