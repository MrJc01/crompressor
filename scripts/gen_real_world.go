//go:build ignore

// gen_real_world.go generates realistic test data with highly repetitive patterns.
// This data is used to train a real codebook and demonstrate effective compression.
//
// Usage: go run scripts/gen_real_world.go
package main

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
)

const outputDir = "testdata/real_world"

func main() {
	os.MkdirAll(outputDir, 0755)

	generators := []struct {
		name string
		fn   func() []byte
	}{
		{"go_source", genGoSource},
		{"server_logs", genServerLogs},
		{"config_files", genConfigFiles},
		{"json_data", genJSONData},
		{"binary_headers", genBinaryHeaders},
	}

	totalSize := 0
	for _, g := range generators {
		data := g.fn()
		path := filepath.Join(outputDir, g.name)
		os.MkdirAll(path, 0755)

		// Split into multiple files for realistic training
		chunkSize := len(data) / 5
		for i := 0; i < 5; i++ {
			start := i * chunkSize
			end := start + chunkSize
			if i == 4 {
				end = len(data)
			}
			fname := filepath.Join(path, fmt.Sprintf("part_%03d.dat", i))
			os.WriteFile(fname, data[start:end], 0644)
			totalSize += end - start
		}
	}

	fmt.Printf("╔═══════════════════════════════════════════════╗\n")
	fmt.Printf("║       CROM REAL-WORLD DATA GENERATOR          ║\n")
	fmt.Printf("╠═══════════════════════════════════════════════╣\n")
	fmt.Printf("║  Output:    %-33s ║\n", outputDir)
	fmt.Printf("║  Total Size: %-32s ║\n", formatSize(totalSize))
	fmt.Printf("║  Categories: 5 (Go, Logs, Config, JSON, Bin) ║\n")
	fmt.Printf("╚═══════════════════════════════════════════════╝\n")
}

func formatSize(bytes int) string {
	if bytes >= 1024*1024 {
		return fmt.Sprintf("%.2f MB", float64(bytes)/(1024*1024))
	}
	return fmt.Sprintf("%.2f KB", float64(bytes)/1024)
}

// genGoSource generates repetitive Go source code patterns
func genGoSource() []byte {
	templates := []string{
		"package main\n\nimport (\n\t\"fmt\"\n\t\"os\"\n)\n\nfunc main() {\n\tfmt.Println(\"Hello, World!\")\n\tos.Exit(0)\n}\n",
		"func handler(w http.ResponseWriter, r *http.Request) {\n\tw.Header().Set(\"Content-Type\", \"application/json\")\n\tw.WriteHeader(http.StatusOK)\n\tjson.NewEncoder(w).Encode(response)\n}\n",
		"type Config struct {\n\tHost     string `json:\"host\"`\n\tPort     int    `json:\"port\"`\n\tDatabase string `json:\"database\"`\n\tTimeout  int    `json:\"timeout\"`\n}\n",
		"if err != nil {\n\tlog.Printf(\"error: %v\", err)\n\treturn nil, fmt.Errorf(\"failed to process: %w\", err)\n}\n",
		"for i := 0; i < len(items); i++ {\n\tresult = append(result, process(items[i]))\n}\nreturn result, nil\n",
		"func TestHandler(t *testing.T) {\n\treq := httptest.NewRequest(\"GET\", \"/api/v1/data\", nil)\n\trec := httptest.NewRecorder()\n\thandler(rec, req)\n\tif rec.Code != 200 {\n\t\tt.Errorf(\"expected 200, got %d\", rec.Code)\n\t}\n}\n",
	}

	var buf strings.Builder
	for buf.Len() < 2*1024*1024 { // 2MB
		buf.WriteString(templates[rand.Intn(len(templates))])
	}
	return []byte(buf.String())
}

// genServerLogs generates highly repetitive HTTP server log lines
func genServerLogs() []byte {
	ips := []string{"192.168.1.100", "10.0.0.42", "172.16.0.1", "192.168.1.200"}
	paths := []string{"/api/v1/users", "/api/v1/data", "/health", "/api/v2/products", "/login"}
	methods := []string{"GET", "POST", "PUT", "DELETE"}
	codes := []string{"200", "201", "404", "500", "302"}

	var buf strings.Builder
	for buf.Len() < 2*1024*1024 { // 2MB
		ip := ips[rand.Intn(len(ips))]
		path := paths[rand.Intn(len(paths))]
		method := methods[rand.Intn(len(methods))]
		code := codes[rand.Intn(len(codes))]
		buf.WriteString(fmt.Sprintf("[2024-01-15 12:34:56] %s %s %s %s 128ms \"Mozilla/5.0\"\n", ip, method, path, code))
	}
	return []byte(buf.String())
}

// genConfigFiles generates repetitive YAML/TOML-style configuration
func genConfigFiles() []byte {
	templates := []string{
		"[server]\nhost = \"0.0.0.0\"\nport = 8080\nread_timeout = 30\nwrite_timeout = 30\nmax_connections = 1000\n\n",
		"[database]\ndriver = \"postgres\"\nhost = \"localhost\"\nport = 5432\nname = \"production\"\nuser = \"admin\"\npassword = \"secret\"\nmax_idle = 10\nmax_open = 100\n\n",
		"[logging]\nlevel = \"info\"\nformat = \"json\"\noutput = \"stdout\"\nrotate = true\nmax_size = 100\nmax_backups = 5\n\n",
		"[cache]\ndriver = \"redis\"\nhost = \"localhost\"\nport = 6379\nttl = 3600\nmax_memory = \"256mb\"\neviction = \"lru\"\n\n",
	}

	var buf strings.Builder
	for buf.Len() < 2*1024*1024 { // 2MB
		buf.WriteString(templates[rand.Intn(len(templates))])
	}
	return []byte(buf.String())
}

// genJSONData generates repetitive JSON API responses
func genJSONData() []byte {
	template := `{"id":%d,"name":"user_%d","email":"user%d@example.com","role":"admin","active":true,"created_at":"2024-01-15T12:00:00Z","metadata":{"login_count":42,"last_ip":"192.168.1.100"}}` + "\n"

	var buf strings.Builder
	id := 1
	for buf.Len() < 2*1024*1024 { // 2MB
		buf.WriteString(fmt.Sprintf(template, id, id, id))
		id++
		if id > 100 {
			id = 1 // Reset to create repetition
		}
	}
	return []byte(buf.String())
}

// genBinaryHeaders generates repetitive binary-like structures (ELF/PE headers)
func genBinaryHeaders() []byte {
	// Simulate common binary file patterns
	elfMagic := []byte{0x7f, 'E', 'L', 'F', 2, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	nullPad := make([]byte, 112) // Common null padding in binaries

	var data []byte
	for len(data) < 2*1024*1024 { // 2MB
		data = append(data, elfMagic...)
		data = append(data, nullPad...)
	}
	return data
}
