package delta

import (
	"bytes"
)

const (
	OpEqual  byte = 0
	OpInsert byte = 1
	OpDelete byte = 2
)

// Diff creates a minimal edit script turning 'pattern' into 'original'.
// It uses a simple dynamic programming approach for Levenshtein/LCS,
// optimized for tiny chunks (< 512 bytes).
// Format: sequences of [Opcode] [Length uint16] [Optional Data...]
func Diff(original, pattern []byte) []byte {
	lenO, lenP := len(original), len(pattern)
	
	// Create dynamic programming table
	dp := make([][]int, lenO+1)
	for i := range dp {
		dp[i] = make([]int, lenP+1)
		dp[i][0] = i
	}
	for j := 0; j <= lenP; j++ {
		dp[0][j] = j
	}

	for i := 1; i <= lenO; i++ {
		for j := 1; j <= lenP; j++ {
			if original[i-1] == pattern[j-1] {
				dp[i][j] = dp[i-1][j-1]
			} else {
				m := dp[i-1][j] + 1     // Insert
				if dp[i][j-1]+1 < m {   // Delete
					m = dp[i][j-1] + 1
				}
				if dp[i-1][j-1]+1 < m { // Substitute (Delete + Insert)
					m = dp[i-1][j-1] + 2 
				}
				dp[i][j] = m
			}
		}
	}

	// Backtrack to find edits
	var ops []byte // We will build operations backwards then reverse
	var data []byte

	i, j := lenO, lenP
	for i > 0 || j > 0 {
		if i > 0 && j > 0 && original[i-1] == pattern[j-1] {
			ops = append(ops, OpEqual)
			i--
			j--
		} else if i > 0 && j > 0 && dp[i][j] == dp[i-1][j-1]+2 {
			// Substitution = Delete + Insert
			ops = append(ops, OpDelete, OpInsert)
			data = append(data, original[i-1])
			i--
			j--
		} else if i > 0 && (j == 0 || dp[i][j] == dp[i-1][j]+1) {
			// Insert
			ops = append(ops, OpInsert)
			data = append(data, original[i-1])
			i--
		} else {
			// Delete
			ops = append(ops, OpDelete)
			j--
		}
	}

	// Reverse ops and data
	for k := 0; k < len(ops)/2; k++ {
		ops[k], ops[len(ops)-1-k] = ops[len(ops)-1-k], ops[k]
	}
	for k := 0; k < len(data)/2; k++ {
		data[k], data[len(data)-1-k] = data[len(data)-1-k], data[k]
	}

	// Run-length encode the operations
	var script bytes.Buffer
	dataIdx := 0

	for k := 0; k < len(ops); {
		op := ops[k]
		count := 1
		for k+count < len(ops) && ops[k+count] == op && count < 255 {
			count++
		}

		script.WriteByte(op)
		script.WriteByte(byte(count))

		if op == OpInsert {
			script.Write(data[dataIdx : dataIdx+count])
			dataIdx += count
		}
		k += count
	}

	return script.Bytes()
}

// ApplyPatch constructs 'original' from 'pattern' using the edit script.
func ApplyPatch(pattern, script []byte) ([]byte, error) {
	var original bytes.Buffer
	patIdx := 0
	scrIdx := 0

	for scrIdx < len(script) {
		op := script[scrIdx]
		count := int(script[scrIdx+1])
		scrIdx += 2

		switch op {
		case OpEqual:
			end := patIdx + count
			if end > len(pattern) {
				return nil, bytes.ErrTooLarge
			}
			original.Write(pattern[patIdx:end])
			patIdx += count
		case OpInsert:
			original.Write(script[scrIdx : scrIdx+count])
			scrIdx += count
		case OpDelete:
			patIdx += count
		}
	}

	return original.Bytes(), nil
}
