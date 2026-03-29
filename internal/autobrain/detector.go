package autobrain

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/MrJc01/crompressor/internal/entropy"
)

type DetectionResult struct {
	Category   string
	Confidence float64
	Entropy    float64
	MagicHint  string
}

// Common Magic Bytes
var (
	magicPNG   = []byte{0x89, 0x50, 0x4E, 0x47}
	magicZIP   = []byte{0x50, 0x4B, 0x03, 0x04}
	magicGZIP  = []byte{0x1F, 0x8B}
	magicBMP   = []byte{0x42, 0x4D}
	magicJPEG  = []byte{0xFF, 0xD8, 0xFF}
	magicGIF87 = []byte{0x47, 0x49, 0x46, 0x38, 0x37, 0x61}
	magicGIF89 = []byte{0x47, 0x49, 0x46, 0x38, 0x39, 0x61}
	magicTIFF1 = []byte{0x49, 0x49, 0x2A, 0x00}
	magicTIFF2 = []byte{0x4D, 0x4D, 0x00, 0x2A}
	magicELF   = []byte{0x7F, 0x45, 0x4C, 0x46}
	magicPDF   = []byte{0x25, 0x50, 0x44, 0x46}
)

func DetectFormat(filePath string) (*DetectionResult, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("detect format: %w", err)
	}
	defer f.Close()

	// Analyze the first 8KB
	eScore, buf, err := entropy.Analyze(f, 8192)
	if err != nil {
		return nil, fmt.Errorf("entropy analysis: %w", err)
	}

	if len(buf) == 0 {
		return &DetectionResult{
			Category:   "empty",
			Confidence: 1.0,
			Entropy:    0,
			MagicHint:  "none",
		}, nil
	}

	res := &DetectionResult{
		Entropy: eScore,
	}

	// 1. Check Magic Bytes
	if bytes.HasPrefix(buf, magicPNG) {
		res.Category = "compressed"
		res.MagicHint = "png"
		res.Confidence = 1.0
		return res, nil
	}
	if bytes.HasPrefix(buf, magicZIP) {
		res.Category = "compressed"
		res.MagicHint = "zip"
		res.Confidence = 1.0
		return res, nil
	}
	if bytes.HasPrefix(buf, magicGZIP) {
		res.Category = "compressed"
		res.MagicHint = "gzip"
		res.Confidence = 1.0
		return res, nil
	}
	if bytes.HasPrefix(buf, magicJPEG) {
		res.Category = "compressed"
		res.MagicHint = "jpeg"
		res.Confidence = 1.0
		return res, nil
	}
	if bytes.HasPrefix(buf, magicGIF87) || bytes.HasPrefix(buf, magicGIF89) {
		res.Category = "compressed"
		res.MagicHint = "gif"
		res.Confidence = 1.0
		return res, nil
	}
	if len(buf) >= 12 && string(buf[0:4]) == "RIFF" && string(buf[8:12]) == "WEBP" {
		res.Category = "compressed"
		res.MagicHint = "webp"
		res.Confidence = 1.0
		return res, nil
	}
	if bytes.HasPrefix(buf, magicBMP) {
		res.Category = "raw_image"
		res.MagicHint = "bmp"
		res.Confidence = 0.9
		return res, nil
	}
	if bytes.HasPrefix(buf, magicTIFF1) || bytes.HasPrefix(buf, magicTIFF2) {
		res.Category = "raw_image"
		res.MagicHint = "tiff"
		res.Confidence = 0.9
		return res, nil
	}
	if bytes.HasPrefix(buf, magicELF) {
		res.Category = "binary"
		res.MagicHint = "elf"
		res.Confidence = 0.9
		return res, nil
	}
	if bytes.HasPrefix(buf, magicPDF) {
		res.Category = "binary"
		res.MagicHint = "pdf"
		res.Confidence = 0.9
		return res, nil
	}

	// 2. Text Analysis (heuristics)
	content := string(buf)
	upperContent := strings.ToUpper(content)

	isSQL := strings.Contains(upperContent, "SELECT ") || 
		strings.Contains(upperContent, "INSERT INTO ") || 
		strings.Contains(upperContent, "CREATE TABLE ")

	isCode := strings.Contains(content, "func ") || 
		strings.Contains(content, "class ") || 
		strings.Contains(content, "def ") || 
		strings.Contains(content, "package ") ||
		strings.Contains(content, "import ") ||
		strings.Contains(content, "function ")

	isSVG := strings.Contains(content, "<svg") || strings.Contains(upperContent, "XML")

	if isSVG {
		res.Category = "raw_image"
		res.MagicHint = "svg"
		res.Confidence = 0.8
		return res, nil
	}

	if isSQL {
		res.Category = "text_sql"
		res.MagicHint = "sql"
		res.Confidence = 0.8
		return res, nil
	}

	if isCode {
		res.Category = "text_code"
		res.MagicHint = "code"
		res.Confidence = 0.7
		return res, nil
	}

	// Logs and generic text (usually mostly printable ASCII)
	printableChars := 0
	for _, b := range buf {
		if (b >= 32 && b <= 126) || b == '\n' || b == '\r' || b == '\t' {
			printableChars++
		}
	}
	printableRatio := float64(printableChars) / float64(len(buf))

	if printableRatio > 0.85 {
		res.Category = "text_logs"
		res.MagicHint = "text"
		res.Confidence = 0.7
		return res, nil
	}

	// 3. Fallback based on entropy
	res.MagicHint = "unknown"
	if eScore > 6.5 {
		res.Category = "binary"
		res.Confidence = 0.5
	} else {
		// Mid/Low entropy but not mostly text -> generic raw binary data
		res.Category = "binary"
		res.Confidence = 0.3
	}

	return res, nil
}
