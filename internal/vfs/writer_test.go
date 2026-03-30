package vfs

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/MrJc01/crompressor/pkg/format"
)

func TestV9_AppendMutation_WAL(t *testing.T) {
	tmpDir := t.TempDir()
	cromFile := filepath.Join(tmpDir, "test.crom")

	// Create a dummy format.Version9 file
	f, err := os.Create(cromFile)
	if err != nil {
		t.Fatal(err)
	}
	f.Write([]byte("CROM"))                  // Magic
	f.Write([]byte{byte(format.Version9), 0}) // Version9
	
	// Pad fake header to base size
	pad := make([]byte, format.HeaderSizeV8-6)
	f.Write(pad)
	f.Close()

	// Initialize WAL
	wal := NewWriteAheadLog(cromFile)

	// Simulate multiple rapid FUSE writes
	wal.Append([]byte("Hello "), 0)
	wal.Append([]byte("World"), 6)
	wal.Append([]byte("!"), 11)

	// Check if buffer accumulated them (it should, without immediate flush)
	wal.mu.Lock()
	if wal.buffer.Len() != 12 {
		t.Fatalf("Expected buffer length 12, got %d", wal.buffer.Len())
	}
	wal.mu.Unlock()

	// Wait for tick or force close
	wal.Close()

	// Verify the .crom file now has the mutating header at the end
	data, err := os.ReadFile(cromFile)
	if err != nil {
		t.Fatal(err)
	}

	// Payload Should be at the very end 
	if !bytes.HasSuffix(data, []byte("Hello World!")) {
		t.Fatalf("WAL did not append mutation efficiently. File ends with: %s", data[len(data)-12:])
	}

	// Read backwards 16 bytes from payload to find CMUT magic
	headerStart := len(data) - 12 - format.V9MutationHeaderSize
	if headerStart < 0 {
		t.Fatal("File too small to contain header")
	}

	magic := data[headerStart : headerStart+4]
	if string(magic) != "CMUT" {
		t.Fatalf("Expected magic 'CMUT', got '%s'", string(magic))
	}
}
