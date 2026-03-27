package codebook

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
)

// testCodewordSize and testCodewordCount match gen_mini_codebook.go defaults.
const (
	testCodewordSize  = 128
	testCodewordCount = 256 // Smaller for tests (vs 8192 in gen script)
	testSeed          = 42
)

// createTestCodebook generates a temporary .cromdb file for testing.
func createTestCodebook(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "test.cromdb")

	rng := rand.New(rand.NewSource(testSeed))
	dataSize := testCodewordSize * testCodewordCount
	codewordData := make([]byte, dataSize)
	rng.Read(codewordData)

	buildHash := sha256.Sum256(codewordData)

	header := make([]byte, HeaderSize)
	copy(header[0:MagicSize], MagicString)
	binary.LittleEndian.PutUint16(header[6:8], Version1)
	binary.LittleEndian.PutUint16(header[8:10], testCodewordSize)
	binary.LittleEndian.PutUint64(header[10:18], testCodewordCount)
	binary.LittleEndian.PutUint64(header[18:26], HeaderSize)
	copy(header[26:58], buildHash[:])

	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	f.Write(header)
	f.Write(codewordData)
	f.Close()

	return path
}

func TestParseHeader_Valid(t *testing.T) {
	path := createTestCodebook(t)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	h, err := ParseHeader(data)
	if err != nil {
		t.Fatalf("ParseHeader failed: %v", err)
	}

	if string(h.Magic[:]) != MagicString {
		t.Errorf("magic: got %q, want %q", string(h.Magic[:]), MagicString)
	}
	if h.Version != Version1 {
		t.Errorf("version: got %d, want %d", h.Version, Version1)
	}
	if h.CodewordSize != testCodewordSize {
		t.Errorf("codeword size: got %d, want %d", h.CodewordSize, testCodewordSize)
	}
	if h.CodewordCount != testCodewordCount {
		t.Errorf("codeword count: got %d, want %d", h.CodewordCount, testCodewordCount)
	}
	if h.DataOffset != HeaderSize {
		t.Errorf("data offset: got %d, want %d", h.DataOffset, HeaderSize)
	}
}

func TestParseHeader_InvalidMagic(t *testing.T) {
	data := make([]byte, HeaderSize)
	copy(data[0:6], "BADMAG")

	_, err := ParseHeader(data)
	if err == nil {
		t.Fatal("expected error for invalid magic")
	}
}

func TestParseHeader_TooShort(t *testing.T) {
	data := make([]byte, 100) // < HeaderSize
	_, err := ParseHeader(data)
	if err == nil {
		t.Fatal("expected error for short data")
	}
}

func TestHeaderSerializeRoundtrip(t *testing.T) {
	h := &Header{
		Version:       Version1,
		CodewordSize:  testCodewordSize,
		CodewordCount: testCodewordCount,
		DataOffset:    HeaderSize,
	}
	copy(h.Magic[:], MagicString)
	h.BuildHash = sha256.Sum256([]byte("test"))

	buf := h.Serialize()
	if len(buf) != HeaderSize {
		t.Fatalf("serialized length: got %d, want %d", len(buf), HeaderSize)
	}

	h2, err := ParseHeader(buf)
	if err != nil {
		t.Fatalf("ParseHeader on serialized data failed: %v", err)
	}

	if h2.Version != h.Version {
		t.Errorf("roundtrip version mismatch")
	}
	if h2.CodewordSize != h.CodewordSize {
		t.Errorf("roundtrip codeword size mismatch")
	}
	if h2.CodewordCount != h.CodewordCount {
		t.Errorf("roundtrip codeword count mismatch")
	}
	if h2.BuildHash != h.BuildHash {
		t.Errorf("roundtrip build hash mismatch")
	}
}

func TestOpen_ValidCodebook(t *testing.T) {
	path := createTestCodebook(t)

	reader, err := Open(path)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer reader.Close()

	if reader.CodewordCount() != testCodewordCount {
		t.Errorf("codeword count: got %d, want %d", reader.CodewordCount(), testCodewordCount)
	}
	if reader.CodewordSize() != testCodewordSize {
		t.Errorf("codeword size: got %d, want %d", reader.CodewordSize(), testCodewordSize)
	}
}

func TestOpen_NonexistentFile(t *testing.T) {
	_, err := Open("/tmp/nonexistent.cromdb")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestLookup_ValidID(t *testing.T) {
	path := createTestCodebook(t)
	reader, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()

	// Reproduce the exact same RNG to verify codeword content
	rng := rand.New(rand.NewSource(testSeed))
	expectedData := make([]byte, testCodewordSize*testCodewordCount)
	rng.Read(expectedData)

	// Check first, middle, and last codewords
	ids := []uint64{0, testCodewordCount / 2, testCodewordCount - 1}
	for _, id := range ids {
		cw, err := reader.Lookup(id)
		if err != nil {
			t.Fatalf("Lookup(%d) failed: %v", id, err)
		}
		if len(cw) != testCodewordSize {
			t.Errorf("Lookup(%d): length %d, want %d", id, len(cw), testCodewordSize)
		}

		expectedStart := id * testCodewordSize
		expected := expectedData[expectedStart : expectedStart+testCodewordSize]
		if !bytes.Equal(cw, expected) {
			t.Errorf("Lookup(%d): data mismatch at first byte: got 0x%02x, want 0x%02x",
				id, cw[0], expected[0])
		}
	}
}

func TestLookup_OutOfBounds(t *testing.T) {
	path := createTestCodebook(t)
	reader, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()

	_, err = reader.Lookup(testCodewordCount) // ID == count → OOB
	if err == nil {
		t.Fatal("expected error for out-of-bounds lookup")
	}

	_, err = reader.Lookup(testCodewordCount + 1000)
	if err == nil {
		t.Fatal("expected error for far out-of-bounds lookup")
	}
}

func TestLookup_AllCodewords(t *testing.T) {
	path := createTestCodebook(t)
	reader, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()

	// Verify all codewords are accessible
	for id := uint64(0); id < testCodewordCount; id++ {
		cw, err := reader.Lookup(id)
		if err != nil {
			t.Fatalf("Lookup(%d) failed: %v", id, err)
		}
		if len(cw) != testCodewordSize {
			t.Fatalf("Lookup(%d): wrong length %d", id, len(cw))
		}
	}
}

func BenchmarkLookup(b *testing.B) {
	// Create a temp codebook
	dir := b.TempDir()
	path := filepath.Join(dir, "bench.cromdb")

	rng := rand.New(rand.NewSource(testSeed))
	dataSize := testCodewordSize * testCodewordCount
	codewordData := make([]byte, dataSize)
	rng.Read(codewordData)
	buildHash := sha256.Sum256(codewordData)

	header := make([]byte, HeaderSize)
	copy(header[0:MagicSize], MagicString)
	binary.LittleEndian.PutUint16(header[6:8], Version1)
	binary.LittleEndian.PutUint16(header[8:10], testCodewordSize)
	binary.LittleEndian.PutUint64(header[10:18], testCodewordCount)
	binary.LittleEndian.PutUint64(header[18:26], HeaderSize)
	copy(header[26:58], buildHash[:])

	f, _ := os.Create(path)
	f.Write(header)
	f.Write(codewordData)
	f.Close()

	reader, _ := Open(path)
	defer reader.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		id := uint64(i % testCodewordCount)
		reader.Lookup(id)
	}
}
