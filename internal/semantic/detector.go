package semantic

import (
	"bytes"
)

// DetectHeuristicExtension analyzes the first few bytes of a file to guess its content type
// for semantic chunking. Returns "JSON", "LINES", "JSONL", "ELF", "ZIP", or "UNKNOWN".
func DetectHeuristicExtension(sample []byte) string {
	if len(sample) == 0 {
		return "UNKNOWN"
	}

	// 1. Check Magic Bytes for binaries
	if len(sample) >= 4 {
		if bytes.HasPrefix(sample, []byte{0x7f, 'E', 'L', 'F'}) {
			return "ELF"
		}
		if bytes.HasPrefix(sample, []byte{'P', 'K', 0x03, 0x04}) {
			return "ZIP"
		}
		if bytes.HasPrefix(sample, []byte{0x89, 'P', 'N', 'G'}) {
			return "PNG"
		}
	}

	// 2. Check for JSON / JSONL structures
	// Heuristic: looks for '{' at the beginning (ignoring whitespace).
	isJSON := false
	hasNewlines := false
	for _, b := range sample {
		if b == '{' || b == '[' {
			isJSON = true
			break
		} else if b != ' ' && b != '\t' && b != '\n' && b != '\r' {
			break
		}
	}

	for _, b := range sample {
		if b == '\n' {
			hasNewlines = true
			break
		}
	}

	if isJSON {
		if hasNewlines && bytes.Contains(sample, []byte("}\n{")) {
			return "JSONL" // JSON Lines format
		}
		return "JSON"
	}

	// 3. Fallback to LINE-based if it looks like textual code/logs
	// If it contains more than 10 newlines in the first 8KB, we assume it's line-based text.
	newlineCount := 0
	nonPrintable := 0
	for _, b := range sample {
		if b == '\n' {
			newlineCount++
		}
		if b < 32 && b != '\n' && b != '\r' && b != '\t' {
			nonPrintable++
		}
	}

	if newlineCount > 5 && nonPrintable < len(sample)/10 {
		return "LINES"
	}

	return "UNKNOWN"
}
