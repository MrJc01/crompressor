package cromlib

import (
	"bytes"
	"math/rand"
	"os"
	"path/filepath"
	"testing"

	"github.com/MrJc01/crompressor/internal/delta"
	"github.com/MrJc01/crompressor/internal/trainer"
)

// trainBrain builds a 4 KB brain over data, like two users sharing a Cérebro.
func trainBrain(t *testing.T, dir string, data []byte) string {
	t.Helper()
	trainDir := filepath.Join(dir, "train")
	os.MkdirAll(trainDir, 0755)
	if err := os.WriteFile(filepath.Join(trainDir, "v1.bin"), data, 0644); err != nil {
		t.Fatal(err)
	}
	opts := trainer.DefaultTrainOptions()
	opts.InputDir = trainDir
	opts.OutputPath = filepath.Join(dir, "brain.cromdb")
	opts.ChunkSize = 4096
	opts.MaxCodewords = 20000
	if _, err := trainer.Train(opts); err != nil {
		t.Fatal(err)
	}
	return opts.OutputPath
}

func randomBytes(n int) []byte {
	data := make([]byte, n)
	rand.New(rand.NewSource(7)).Read(data)
	return data
}

// packRoundTrip returns the packed size and fails unless the output is identical.
func packRoundTrip(t *testing.T, dir, brain string, data []byte) int64 {
	t.Helper()
	in := filepath.Join(dir, "in.bin")
	packed := filepath.Join(dir, "in.crom")
	out := filepath.Join(dir, "out.bin")
	if err := os.WriteFile(in, data, 0644); err != nil {
		t.Fatal(err)
	}
	opts := DefaultPackOptions()
	if _, err := Pack(in, packed, brain, opts); err != nil {
		t.Fatal(err)
	}
	if err := Unpack(packed, out, brain, DefaultUnpackOptions()); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, data) {
		t.Fatal("restored file differs from the original")
	}
	info, err := os.Stat(packed)
	if err != nil {
		t.Fatal(err)
	}
	return info.Size()
}

// Random data proves the savings come from the brain and not from zstd.
func TestSharedBrainSendsOnlyReferences(t *testing.T) {
	dir := t.TempDir()
	data := randomBytes(2 * 1024 * 1024)
	brain := trainBrain(t, dir, data)

	const metadataPerChunk = 40 // table entry plus a margin for the header
	limit := int64(len(data)/4096)*metadataPerChunk + 1024

	if size := packRoundTrip(t, dir, brain, data); size > limit {
		t.Errorf("identical file: %d bytes, want at most %d", size, limit)
	}

	shifted := append([]byte("X"), data...)
	if size := packRoundTrip(t, dir, brain, shifted); size > limit+8192 {
		t.Errorf("file with an inserted byte: %d bytes, want at most %d", size, limit+8192)
	}

	edited := append([]byte(nil), data...)
	edited[len(edited)/2] ^= 0xff
	if size := packRoundTrip(t, dir, brain, edited); size > limit+8192 {
		t.Errorf("file with an edited byte: %d bytes, want at most %d", size, limit+8192)
	}
}

// Blocks hold 16 MB of input whatever the chunk size, so large files must survive.
func TestRoundTripAcrossBlocksWithLargeChunks(t *testing.T) {
	if testing.Short() {
		t.Skip("large file")
	}
	dir := t.TempDir()
	data := randomBytes(BlockSize + BlockSize/2)
	brain := trainBrain(t, dir, data[:1024*1024])
	packRoundTrip(t, dir, brain, data)
}

func TestUnknownDataStillRoundTrips(t *testing.T) {
	dir := t.TempDir()
	brain := trainBrain(t, dir, randomBytes(256*1024))
	text := bytes.Repeat([]byte("linha de log que o cerebro nunca viu\n"), 20000)
	packRoundTrip(t, dir, brain, text)
}

func TestFailedIntegrityLeavesNoOutput(t *testing.T) {
	dir := t.TempDir()
	data := randomBytes(1024 * 1024)
	brain := trainBrain(t, dir, data)
	otherDir := filepath.Join(dir, "other")
	os.MkdirAll(otherDir, 0755)
	otherBrain := trainBrain(t, otherDir, bytes.Repeat(randomBytes(4096*4), 10)[1:])

	packRoundTrip(t, dir, brain, data)
	out := filepath.Join(dir, "wrong.bin")
	if err := Unpack(filepath.Join(dir, "in.crom"), out, otherBrain, DefaultUnpackOptions()); err == nil {
		t.Fatal("unpack with the wrong brain should fail")
	}
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		t.Error("output file must not exist after a failed integrity check")
	}
	if _, err := os.Stat(out + ".tmp"); !os.IsNotExist(err) {
		t.Error("temporary output must be removed after a failed integrity check")
	}
}

func TestEdgeModeRestoresLossyOutput(t *testing.T) {
	dir := t.TempDir()
	data := randomBytes(64 * 1024)
	brain := trainBrain(t, dir, data)
	in := filepath.Join(dir, "edge.bin")
	edited := append([]byte(nil), data...)
	edited[100] ^= 0xff
	os.WriteFile(in, edited, 0644)

	opts := DefaultPackOptions()
	opts.Mode = "edge"
	packed := filepath.Join(dir, "edge.crom")
	if _, err := Pack(in, packed, brain, opts); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "edge.out")
	if err := Unpack(packed, out, brain, DefaultUnpackOptions()); err != nil {
		t.Fatal(err)
	}
	if info, err := os.Stat(out); err != nil || info.Size() != int64(len(data)) {
		t.Fatalf("edge output: %v, %v", info, err)
	}
}

func TestPackBytesWithSharedBrain(t *testing.T) {
	dir := t.TempDir()
	data := randomBytes(1024 * 1024)
	brainData, err := os.ReadFile(trainBrain(t, dir, data))
	if err != nil {
		t.Fatal(err)
	}
	packed, _, err := PackBytes(data, brainData, DefaultPackOptions())
	if err != nil {
		t.Fatal(err)
	}
	if len(packed) > len(data)/50 {
		t.Errorf("packed to %d bytes, want references only", len(packed))
	}
	restored, err := UnpackBytes(packed, brainData, DefaultUnpackOptions())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(restored, data) {
		t.Fatal("restored bytes differ")
	}
}

func TestZeroConfigRestoresWithoutCodebook(t *testing.T) {
	for _, key := range []string{"", "segredo"} {
		dir := t.TempDir()
		var data []byte
		for i := 0; len(data) < 3*1024*1024; i++ {
			data = append(data, []byte("2026-10-30 INFO api GET /v1/items/")...)
			data = append(data, byte('0'+i%10), byte('0'+i%7), '\n')
		}
		in := filepath.Join(dir, "in.log")
		os.WriteFile(in, data, 0644)
		packed := filepath.Join(dir, "in.crom")
		opts := DefaultPackOptions()
		opts.EncryptionKey = key
		if _, err := AutoPack(in, packed, opts); err != nil {
			t.Fatal(err)
		}
		uopts := DefaultUnpackOptions()
		uopts.EncryptionKey = key
		out := filepath.Join(dir, "out.log")
		if err := Unpack(packed, out, "", uopts); err != nil {
			t.Fatalf("key %q: %v", key, err)
		}
		if got, _ := os.ReadFile(out); !bytes.Equal(got, data) {
			t.Fatalf("key %q: restored file differs", key)
		}
	}
}

func TestEmbeddedBrainRestoresWithoutCodebook(t *testing.T) {
	for _, key := range []string{"", "segredo"} {
		dir := t.TempDir()
		data := randomBytes(256 * 1024)
		brain := trainBrain(t, dir, data)
		in := filepath.Join(dir, "in.bin")
		os.WriteFile(in, data, 0644)
		packed := filepath.Join(dir, "in.crom")
		opts := DefaultPackOptions()
		opts.EmbedBrain = true
		opts.EncryptionKey = key
		if _, err := Pack(in, packed, brain, opts); err != nil {
			t.Fatal(err)
		}
		uopts := DefaultUnpackOptions()
		uopts.EncryptionKey = key
		out := filepath.Join(dir, "out.bin")
		if err := Unpack(packed, out, "", uopts); err != nil {
			t.Fatalf("key %q: %v", key, err)
		}
		if got, _ := os.ReadFile(out); !bytes.Equal(got, data) {
			t.Fatalf("key %q: restored file differs", key)
		}
	}
}

// Incompressible data falls back to passthrough, which must stay encrypted.
func TestEncryptedPassthroughIsEncryptedAndRestores(t *testing.T) {
	dir := t.TempDir()
	data := randomBytes(256 * 1024)
	brain := trainBrain(t, dir, randomBytes(64*1024))
	in := filepath.Join(dir, "in.bin")
	os.WriteFile(in, data, 0644)

	pack := map[string]func(string, PackOptions) error{
		"brain":    func(p string, o PackOptions) error { _, err := Pack(in, p, brain, o); return err },
		"noBrain":  func(p string, o PackOptions) error { _, err := Pack(in, p, "", o); return err },
		"autoPack": func(p string, o PackOptions) error { _, err := AutoPack(in, p, o); return err },
	}
	for name, run := range pack {
		packed := filepath.Join(dir, name+".crom")
		opts := DefaultPackOptions()
		opts.EncryptionKey = "segredo"
		if err := run(packed, opts); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		raw, _ := os.ReadFile(packed)
		if bytes.Contains(raw, data[1000:1064]) {
			t.Errorf("%s: plaintext found in an encrypted file", name)
		}
		uopts := DefaultUnpackOptions()
		uopts.EncryptionKey = "segredo"
		out := filepath.Join(dir, name+".out")
		if err := Unpack(packed, out, brain, uopts); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if got, _ := os.ReadFile(out); !bytes.Equal(got, data) {
			t.Fatalf("%s: restored file differs", name)
		}
	}
}

func TestPackStreamWithSharedBrain(t *testing.T) {
	dir := t.TempDir()
	data := randomBytes(1024 * 1024)
	brain := trainBrain(t, dir, data)
	shifted := append([]byte("X"), data...)

	packed := filepath.Join(dir, "s.crom")
	if _, err := PackStream(bytes.NewReader(shifted), packed, brain, DefaultPackOptions()); err != nil {
		t.Fatal(err)
	}
	if info, _ := os.Stat(packed); info.Size() > int64(len(data)/50) {
		t.Errorf("packed to %d bytes, want references only", info.Size())
	}
	out := filepath.Join(dir, "s.out")
	if err := Unpack(packed, out, brain, DefaultUnpackOptions()); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(out); !bytes.Equal(got, shifted) {
		t.Fatal("restored file differs")
	}

	noBrain := filepath.Join(dir, "n.crom")
	if _, err := PackStream(bytes.NewReader(shifted), noBrain, "", DefaultPackOptions()); err != nil {
		t.Fatal(err)
	}
	if err := Unpack(noBrain, out, "", DefaultUnpackOptions()); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(out); !bytes.Equal(got, shifted) {
		t.Fatal("restored file differs without a brain")
	}
}

func TestMaxCompressionRestoresWithGoDecoder(t *testing.T) {
	if !delta.MaxCompressionAvailable {
		t.Skip("build without CGO")
	}
	dir := t.TempDir()
	base := bytes.Repeat([]byte("2026-10-30 INFO api GET /v1/items 200\n"), 4000)
	brain := trainBrain(t, dir, base)
	data := append(bytes.Repeat([]byte("2026-10-31 WARN api GET /v1/users 404\n"), 4000), base[:9000]...)
	in := filepath.Join(dir, "in.txt")
	os.WriteFile(in, data, 0644)
	opts := DefaultPackOptions()
	opts.MaxCompression = true
	packed := filepath.Join(dir, "in.crom")
	if _, err := Pack(in, packed, brain, opts); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "out.txt")
	if err := Unpack(packed, out, brain, DefaultUnpackOptions()); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(out); !bytes.Equal(got, data) {
		t.Fatal("restored file differs")
	}
}

func TestFileThatNeedsBrainExplainsIt(t *testing.T) {
	dir := t.TempDir()
	data := randomBytes(64 * 1024)
	brain := trainBrain(t, dir, data)
	packRoundTrip(t, dir, brain, data)
	if err := Unpack(filepath.Join(dir, "in.crom"), filepath.Join(dir, "x"), "", DefaultUnpackOptions()); err == nil {
		t.Fatal("expected an error without the brain")
	}
}
