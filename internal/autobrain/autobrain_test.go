package autobrain

import (
	"os"
	"path/filepath"
	"testing"
)

func createTempFile(t *testing.T, content []byte, ext string) string {
	f, err := os.CreateTemp("", "*"+ext)
	if err != nil {
		t.Fatal(err)
	}
	f.Write(content)
	f.Close()
	return f.Name()
}

func TestDetectFormat(t *testing.T) {
	tests := []struct {
		name     string
		content  []byte
		ext      string
		expected string
	}{
		{
			name:     "Text Logs",
			content:  []byte("2026-03-29 12:00:00 INFO Server started answering requests on port 8080. Everything is fine."),
			ext:      ".log",
			expected: "text_logs",
		},
		{
			name:     "SQL",
			content:  []byte("INSERT INTO users (id, name) VALUES (1, 'John Doe'); SELECT * FROM table;"),
			ext:      ".sql",
			expected: "text_sql",
		},
		{
			name:     "Code",
			content:  []byte("func HelloWorld() {\n\tprintln(\"Hello\")\n}\n"),
			ext:      ".go",
			expected: "text_code",
		},
		{
			name:     "BMP Image",
			content:  append([]byte{0x42, 0x4D}, make([]byte, 100)...), // BMP magic + some empty pixels
			ext:      ".bmp",
			expected: "raw_image",
		},
		{
			name:     "PNG Image",
			content:  append([]byte{0x89, 0x50, 0x4E, 0x47}, []byte("IHDR")...),
			ext:      ".png",
			expected: "compressed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := createTempFile(t, tt.content, tt.ext)
			defer os.Remove(path)

			res, err := DetectFormat(path)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if res.Category != tt.expected {
				t.Errorf("expected %s, got %s (hint=%s)", tt.expected, res.Category, res.MagicHint)
			}
		})
	}
}

func TestBrainRouter(t *testing.T) {
	dir, err := os.MkdirTemp("", "brain_test_dir")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	// Create dummy brain files
	logsBrain := filepath.Join(dir, "brain_logs.cromdb")
	os.WriteFile(logsBrain, []byte("dummy logs codebook"), 0644)

	imgBrain := filepath.Join(dir, "brain_image.cromdb")
	os.WriteFile(imgBrain, []byte("dummy img codebook"), 0644)

	universalBrain := filepath.Join(dir, "brain_universal.cromdb")
	os.WriteFile(universalBrain, []byte("dummy uni codebook"), 0644)

	router, err := NewBrainRouter(dir)
	if err != nil {
		t.Fatalf("NewBrainRouter error: %v", err)
	}

	t.Run("Select Log Brain", func(t *testing.T) {
		logFile := createTempFile(t, []byte("127.0.0.1 - - [10/Oct/2000] \"GET / HTTP/1.0\" 200"), ".log")
		defer os.Remove(logFile)

		brain, res, err := router.SelectBrain(logFile)
		if err != nil {
			t.Fatal(err)
		}
		if res.Category != "text_logs" {
			t.Errorf("Expected category text_logs, got %s", res.Category)
		}
		if brain != logsBrain {
			t.Errorf("Expected brain %s, got %s", logsBrain, brain)
		}
	})

	t.Run("Select Magic Brain", func(t *testing.T) {
		bmpFile := createTempFile(t, append(magicBMP, make([]byte, 10)...), ".bmp")
		defer os.Remove(bmpFile)

		brain, res, err := router.SelectBrain(bmpFile)
		if err != nil {
			t.Fatal(err)
		}
		if res.Category != "raw_image" {
			t.Errorf("Expected category raw_image, got %s", res.Category)
		}
		if brain != imgBrain {
			t.Errorf("Expected brain %s, got %s", imgBrain, brain)
		}
	})
}
