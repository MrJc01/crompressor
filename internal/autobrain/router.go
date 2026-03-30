package autobrain

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type BrainRouter struct {
	brainDir string
	mapping  map[string]string // maps Category to brain path
}

func NewBrainRouter(dir string) (*BrainRouter, error) {
	router := &BrainRouter{
		brainDir: dir,
		mapping:  make(map[string]string),
	}

	// 1. Check if the directory exists
	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			// Instead of failing entirely, create it dynamically to be nice to new users
			err = os.MkdirAll(dir, 0755)
			if err != nil {
				return nil, fmt.Errorf("could not create brain dir: %w", err)
			}
		} else {
			return nil, err
		}
	} else if !info.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", dir)
	}

	// 2. Load brains.json mapping if it exists
	configPath := filepath.Join(dir, "brains.json")
	if configBytes, err := os.ReadFile(configPath); err == nil {
		if err := json.Unmarshal(configBytes, &router.mapping); err != nil {
			return nil, fmt.Errorf("failed to parse brains.json: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("error reading brains.json: %w", err)
	}

	// 3. Fallback to Auto-Discovery based on filenames
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading brain dir: %w", err)
	}

	for _, f := range files {
		if f.IsDir() || !strings.HasSuffix(f.Name(), ".cromdb") {
			continue
		}

		// Example auto-mapping rules:
		// "brain_logs.cromdb" -> text_logs
		// "brain_sql.cromdb" -> text_sql
		// "brain_image.cromdb" or "brain_bmp.cromdb" -> raw_image
		// "brain_universal.cromdb" -> universal
		name := strings.ToLower(f.Name())
		path := filepath.Join(dir, f.Name())

		if strings.Contains(name, "log") && router.mapping["text_logs"] == "" {
			router.mapping["text_logs"] = path
		} else if strings.Contains(name, "sql") && router.mapping["text_sql"] == "" {
			router.mapping["text_sql"] = path
		} else if strings.Contains(name, "code") && router.mapping["text_code"] == "" {
			router.mapping["text_code"] = path
		} else if (strings.Contains(name, "img") || strings.Contains(name, "image") || strings.Contains(name, "bmp") || strings.Contains(name, "tiff") || strings.Contains(name, "svg")) && router.mapping["raw_image"] == "" {
			router.mapping["raw_image"] = path
		} else if strings.Contains(name, "universal") && router.mapping["universal"] == "" {
			router.mapping["universal"] = path
		}
	}

	return router, nil
}

// SelectBrain analyzes the file and routes it to the correct codebook.
func (r *BrainRouter) SelectBrain(filePath string) (string, *DetectionResult, error) {
	det, err := DetectFormat(filePath)
	if err != nil {
		return "", nil, err
	}

	// Domain-aware routing: reject image brains for text data and vice-versa
	isTextCategory := det.Category == "text_logs" || det.Category == "text_sql" || det.Category == "text_code"
	isImageCategory := det.Category == "raw_image"

	// Find expert
	brainPath, ok := r.mapping[det.Category]
	if ok && brainPath != "" {
		if _, err := os.Stat(brainPath); err == nil {
			return brainPath, det, nil
		}
	}

	// Try hints
	if det.MagicHint != "unknown" && det.MagicHint != "none" {
		hintPath, ok := r.mapping[det.MagicHint]
		if ok && hintPath != "" {
			if _, err := os.Stat(hintPath); err == nil {
				return hintPath, det, nil
			}
		}
	}

	// Fallback to universal (but NOT if it's an image brain being used for text)
	universalPath, ok := r.mapping["universal"]
	if ok && universalPath != "" {
		if _, err := os.Stat(universalPath); err == nil {
			return universalPath, det, nil
		}
	}

	// Last resort: try ANY brain, but respect domain boundaries
	for cat, p := range r.mapping {
		if p == "" {
			continue
		}
		// Don't use image brains for text data
		if isTextCategory && cat == "raw_image" {
			continue
		}
		// Don't use text brains for image data
		if isImageCategory && (cat == "text_logs" || cat == "text_sql" || cat == "text_code") {
			continue
		}
		if _, err := os.Stat(p); err == nil {
			return p, det, nil
		}
	}

	return "", det, fmt.Errorf("no valid codebooks found in %s for category '%s' (use --auto-brain or train a domain-specific brain)", r.brainDir, det.Category)
}
