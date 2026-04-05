package merkle

import (
	"reflect"
	"testing"
)

func TestBuildFromChunks_Deterministic(t *testing.T) {
	chunks := [][]byte{
		[]byte("chunk1"),
		[]byte("chunk2"),
		[]byte("chunk3"),
	}

	tree1 := BuildFromChunks(chunks)
	tree2 := BuildFromChunks(chunks)

	if tree1.Root() != tree2.Root() {
		t.Errorf("roots should be deterministic")
	}
}

func TestBuildFromChunks_OddLeaves(t *testing.T) {
	chunks := [][]byte{
		[]byte("1"),
		[]byte("2"),
		[]byte("3"), // odd
	}
	tree := BuildFromChunks(chunks)

	// Since there are 3 leaves, node 0, 1, 2 are the hashes.
	// Node 3 should be duplicated node 2 because of padding to make it even
	if !bytesEqual(tree.Nodes[2], tree.Nodes[3]) {
		t.Errorf("padding failed for odd leaves: %v != %v", tree.Nodes[2], tree.Nodes[3])
	}
}

func TestDiff_Identical(t *testing.T) {
	chunks1 := [][]byte{[]byte("A"), []byte("B"), []byte("C")}
	tree1 := BuildFromChunks(chunks1)
	tree2 := BuildFromChunks(chunks1)

	diffs := tree1.Diff(tree2)
	if len(diffs) != 0 {
		t.Errorf("expected 0 diffs, got %v", diffs)
	}
}

func TestDiff_OneChanged(t *testing.T) {
	chunks1 := [][]byte{[]byte("A"), []byte("B"), []byte("C"), []byte("D")}
	chunks2 := [][]byte{[]byte("A"), []byte("Z"), []byte("C"), []byte("D")} // B changed to Z

	tree1 := BuildFromChunks(chunks1)
	tree2 := BuildFromChunks(chunks2)

	diffs := tree1.Diff(tree2)
	expected := []int{1}

	if !reflect.DeepEqual(diffs, expected) {
		t.Errorf("expected diffs %v, got %v", expected, diffs)
	}
}

func TestDiff_MultipleChanged(t *testing.T) {
	chunks1 := [][]byte{[]byte("A"), []byte("B"), []byte("C"), []byte("D")}
	chunks2 := [][]byte{[]byte("Z"), []byte("B"), []byte("X"), []byte("Y")} // A changed to Z, C to X, D to Y

	tree1 := BuildFromChunks(chunks1)
	tree2 := BuildFromChunks(chunks2)

	diffs := tree1.Diff(tree2)
	expected := []int{0, 2, 3}

	if !reflect.DeepEqual(diffs, expected) {
		t.Errorf("expected diffs %v, got %v", expected, diffs)
	}
}
