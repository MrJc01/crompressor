package merkle

import (
	"crypto/sha256"
)

// MerkleTree represents a binary tree of hashes.
type MerkleTree struct {
	Nodes     [][]byte
	LeafCount int
}

// hashNode returns the sha256 of two concatenated byte slices.
func hashNode(left, right []byte) []byte {
	h := sha256.New()
	h.Write(left)
	h.Write(right)
	return h.Sum(nil)
}

// BuildFromChunks builds the MerkleTree from raw data chunks.
func BuildFromChunks(chunks [][]byte) *MerkleTree {
	if len(chunks) == 0 {
		return &MerkleTree{Nodes: [][]byte{}, LeafCount: 0}
	}

	leaves := make([][]byte, len(chunks))
	for i, chunk := range chunks {
		h := sha256.Sum256(chunk)
		leaves[i] = h[:]
	}

	return BuildFromHashes(leaves)
}

// BuildFromHashes builds the tree from pre-computed leaf hashes.
func BuildFromHashes(leaves [][]byte) *MerkleTree {
	if len(leaves) == 0 {
		return &MerkleTree{Nodes: [][]byte{}, LeafCount: 0}
	}

	leafCount := len(leaves)
	
	// Pad odd number of leaves
	if len(leaves)%2 != 0 {
		leaves = append(leaves, leaves[len(leaves)-1])
	}

	var nodes [][]byte
	nodes = append(nodes, leaves...)

	level := leaves
	for len(level) > 1 {
		var nextLevel [][]byte
		for i := 0; i < len(level); i += 2 {
			if i+1 < len(level) {
				parent := hashNode(level[i], level[i+1])
				nextLevel = append(nextLevel, parent)
			} else {
				// Odd node at the end
				nextLevel = append(nextLevel, level[i])
			}
		}
		
		// Ensure nextLevel has an even number of nodes if it's not the root
		if len(nextLevel)%2 != 0 && len(nextLevel) > 1 {
			nextLevel = append(nextLevel, nextLevel[len(nextLevel)-1])
		}
		
		nodes = append(nodes, nextLevel...)
		level = nextLevel
	}

	return &MerkleTree{
		Nodes:     nodes,
		LeafCount: leafCount,
	}
}

// Root returns the root hash of the tree.
func (t *MerkleTree) Root() [32]byte {
	var root [32]byte
	if len(t.Nodes) == 0 {
		return root
	}
	copy(root[:], t.Nodes[len(t.Nodes)-1])
	return root
}

// Diff compares this tree against another and returns the indices of the leaves that differ.
// It assumes both trees have the same structure/size.
func (t *MerkleTree) Diff(other *MerkleTree) []int {
	if len(t.Nodes) == 0 || len(other.Nodes) == 0 {
		return nil
	}
	if len(t.Nodes) != len(other.Nodes) {
		// Cannot simple-diff if sizes are completely different; just return everything
		res := make([]int, t.LeafCount)
		for i := 0; i < t.LeafCount; i++ {
			res[i] = i
		}
		return res
	}

	// Just compare leaves directly for a simplified diff logic (for small N like block counts)
	// Even though Merkle trees allow log(N) traversals, for N < 1000 comparing leaves is fast enough in Go.
	var diffs []int
	for i := 0; i < t.LeafCount; i++ {
		if !bytesEqual(t.Nodes[i], other.Nodes[i]) {
			diffs = append(diffs, i)
		}
	}
	return diffs
}

func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
