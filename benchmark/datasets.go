package main

import (
	"bytes"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// DatasetInfo holds metadata about a generated dataset.
type DatasetInfo struct {
	Name        string
	Description string
	Path        string
	Size        int64
}

// GenerateAllDatasets creates all benchmark datasets in the given directory.
func GenerateAllDatasets(dir string, maxMB int) ([]DatasetInfo, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	generators := []struct {
		name string
		desc string
		mb   int
		fn   func(path string, sizeMB int) error
	}{
		{"go_source", "Código Go repetitivo com variações", min(10, maxMB), generateGoSource},
		{"json_api", "JSON estruturado com campos repetitivos", min(10, maxMB), generateJSON},
		{"server_logs", "Logs de servidor com timestamps e IPs", min(10, maxMB), generateLogs},
		{"mixed_config", "YAML/TOML configs com seções repetidas", min(5, maxMB), generateConfig},
		{"binary_headers", "Headers binários + padding + structs", min(10, maxMB), generateBinaryHeaders},
		{"polynomial", "Dados fractal (ax²+bx+c mod 256)", 1, generatePolynomial},
		{"high_entropy", "Dados pseudorandom (pior caso)", min(10, maxMB), generateHighEntropy},
		{"real_go_repo", "O próprio código Go do crompressor", 0, generateRealGoRepo},
	}

	var datasets []DatasetInfo
	for _, g := range generators {
		path := filepath.Join(dir, g.name+".bin")
		fmt.Printf("  📦 Gerando dataset: %-20s ", g.name)
		start := time.Now()

		if g.mb == 0 {
			// Special case: real_go_repo
			if err := g.fn(path, 0); err != nil {
				fmt.Printf("❌ %v\n", err)
				continue
			}
		} else {
			if err := g.fn(path, g.mb); err != nil {
				fmt.Printf("❌ %v\n", err)
				continue
			}
		}

		info, err := os.Stat(path)
		if err != nil {
			continue
		}

		fmt.Printf("✅ %s (%.1f MB) em %v\n", g.desc, float64(info.Size())/(1024*1024), time.Since(start))
		datasets = append(datasets, DatasetInfo{
			Name:        g.name,
			Description: g.desc,
			Path:        path,
			Size:        info.Size(),
		})
	}

	return datasets, nil
}

// generateGoSource creates repetitive Go source code.
func generateGoSource(path string, sizeMB int) error {
	rng := rand.New(rand.NewSource(42))
	target := sizeMB * 1024 * 1024
	var buf bytes.Buffer

	templates := []string{
		"package example\n\nimport (\n\t\"fmt\"\n\t\"strings\"\n)\n\n",
		"// %s processes data with O(1) complexity.\nfunc %s(data []byte, size int) ([]byte, error) {\n\tif len(data) == 0 {\n\t\treturn nil, fmt.Errorf(\"%s: empty input\")\n\t}\n\tresult := make([]byte, size)\n\tfor i := range result {\n\t\tresult[i] = data[i %% len(data)]\n\t}\n\treturn result, nil\n}\n\n",
		"type %sConfig struct {\n\tMaxSize     int    `json:\"max_size\"`\n\tBufferSize  int    `json:\"buffer_size\"`\n\tConcurrency int    `json:\"concurrency\"`\n\tName        string `json:\"name\"`\n\tEnabled     bool   `json:\"enabled\"`\n}\n\n",
		"func (c *%sConfig) Validate() error {\n\tif c.MaxSize <= 0 {\n\t\treturn fmt.Errorf(\"invalid max_size: %%d\", c.MaxSize)\n\t}\n\tif c.BufferSize <= 0 {\n\t\tc.BufferSize = 4096\n\t}\n\tif c.Concurrency <= 0 {\n\t\tc.Concurrency = 4\n\t}\n\treturn nil\n}\n\n",
		"// Test%s validates the %s function.\nfunc Test%s(t *testing.T) {\n\ttestCases := []struct {\n\t\tname     string\n\t\tinput    []byte\n\t\texpected int\n\t}{\n\t\t{\"empty\", nil, 0},\n\t\t{\"small\", []byte(\"hello\"), 5},\n\t\t{\"large\", bytes.Repeat([]byte(\"x\"), 1024), 1024},\n\t}\n\tfor _, tc := range testCases {\n\t\tt.Run(tc.name, func(t *testing.T) {\n\t\t\tresult, err := %s(tc.input, tc.expected)\n\t\t\tif err != nil {\n\t\t\t\tt.Fatalf(\"unexpected error: %%v\", err)\n\t\t\t}\n\t\t\tif len(result) != tc.expected {\n\t\t\t\tt.Fatalf(\"got %%d, want %%d\", len(result), tc.expected)\n\t\t\t}\n\t\t})\n\t}\n}\n\n",
	}

	funcNames := []string{
		"Compress", "Decompress", "Encode", "Decode", "Transform",
		"Serialize", "Parse", "Validate", "Process", "Analyze",
		"Extract", "Merge", "Split", "Filter", "Search",
		"Index", "Cache", "Stream", "Buffer", "Queue",
	}

	for buf.Len() < target {
		tmpl := templates[rng.Intn(len(templates))]
		name := funcNames[rng.Intn(len(funcNames))]
		suffix := fmt.Sprintf("%d", rng.Intn(100))
		fullName := name + suffix

		switch strings.Count(tmpl, "%s") {
		case 1:
			buf.WriteString(fmt.Sprintf(tmpl, fullName))
		case 2:
			buf.WriteString(fmt.Sprintf(tmpl, fullName, fullName))
		case 4:
			buf.WriteString(fmt.Sprintf(tmpl, fullName, fullName, fullName, fullName))
		case 5:
			buf.WriteString(fmt.Sprintf(tmpl, fullName, fullName, fullName, fullName, fullName))
		default:
			buf.WriteString(tmpl)
		}
	}

	return os.WriteFile(path, buf.Bytes()[:target], 0644)
}

// generateJSON creates structured JSON data.
func generateJSON(path string, sizeMB int) error {
	rng := rand.New(rand.NewSource(123))
	target := sizeMB * 1024 * 1024
	var buf bytes.Buffer

	buf.WriteString("[\n")
	id := 0
	for buf.Len() < target {
		if id > 0 {
			buf.WriteString(",\n")
		}
		ip := fmt.Sprintf("%d.%d.%d.%d", rng.Intn(256), rng.Intn(256), rng.Intn(256), rng.Intn(256))
		methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH"}
		paths := []string{"/api/v1/users", "/api/v1/orders", "/api/v1/products", "/api/v1/health", "/api/v2/metrics"}
		statuses := []int{200, 201, 301, 400, 401, 403, 404, 500}

		buf.WriteString(fmt.Sprintf(`  {"id": %d, "timestamp": "2026-04-05T%02d:%02d:%02dZ", "ip": "%s", "method": "%s", "path": "%s", "status": %d, "latency_ms": %d, "user_agent": "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36", "request_id": "req-%08x", "response_size": %d}`,
			id,
			rng.Intn(24), rng.Intn(60), rng.Intn(60),
			ip,
			methods[rng.Intn(len(methods))],
			paths[rng.Intn(len(paths))],
			statuses[rng.Intn(len(statuses))],
			rng.Intn(5000),
			rng.Uint32(),
			rng.Intn(100000),
		))
		id++
	}
	buf.WriteString("\n]\n")

	data := buf.Bytes()
	if len(data) > target {
		data = data[:target]
	}
	return os.WriteFile(path, data, 0644)
}

// generateLogs creates server-like log lines.
func generateLogs(path string, sizeMB int) error {
	rng := rand.New(rand.NewSource(456))
	target := sizeMB * 1024 * 1024
	var buf bytes.Buffer

	levels := []string{"INFO", "WARN", "ERROR", "DEBUG", "INFO", "INFO", "INFO", "DEBUG"}
	components := []string{"http.server", "db.postgres", "cache.redis", "auth.oauth2", "queue.kafka", "worker.pool"}
	messages := []string{
		"Request processed successfully in %dms",
		"Connection established to %s:%d",
		"Cache miss for key user:%d, fetching from database",
		"Rate limit exceeded for IP %s, throttling for %ds",
		"Health check passed: CPU=%.1f%% MEM=%.1f%% DISK=%.1f%%",
		"Worker %d completed batch of %d items",
		"Session expired for user %d, redirecting to login",
		"Database query took %dms: SELECT * FROM orders WHERE status = 'pending'",
		"TLS handshake completed with cipher TLS_AES_256_GCM_SHA384",
		"Graceful shutdown initiated, draining %d active connections",
	}

	for buf.Len() < target {
		level := levels[rng.Intn(len(levels))]
		comp := components[rng.Intn(len(components))]
		msg := messages[rng.Intn(len(messages))]

		// Fill format specifiers with random values
		msg = strings.ReplaceAll(msg, "%d", fmt.Sprintf("%d", rng.Intn(10000)))
		msg = strings.ReplaceAll(msg, "%s", fmt.Sprintf("%d.%d.%d.%d", rng.Intn(256), rng.Intn(256), rng.Intn(256), rng.Intn(256)))
		msg = strings.ReplaceAll(msg, "%.1f", fmt.Sprintf("%.1f", rng.Float64()*100))

		line := fmt.Sprintf("2026-04-05T%02d:%02d:%02d.%03dZ [%s] [%s] %s\n",
			rng.Intn(24), rng.Intn(60), rng.Intn(60), rng.Intn(1000),
			level, comp, msg)
		buf.WriteString(line)
	}

	data := buf.Bytes()
	if len(data) > target {
		data = data[:target]
	}
	return os.WriteFile(path, data, 0644)
}

// generateConfig creates YAML-like config data.
func generateConfig(path string, sizeMB int) error {
	rng := rand.New(rand.NewSource(789))
	target := sizeMB * 1024 * 1024
	var buf bytes.Buffer

	services := []string{"api-gateway", "user-service", "order-service", "payment-service", "notification-service", "analytics-engine"}
	envs := []string{"production", "staging", "development"}

	for buf.Len() < target {
		svc := services[rng.Intn(len(services))]
		env := envs[rng.Intn(len(envs))]

		buf.WriteString(fmt.Sprintf(`---
service:
  name: %s
  environment: %s
  version: "%d.%d.%d"
  replicas: %d

server:
  host: "0.0.0.0"
  port: %d
  read_timeout: %ds
  write_timeout: %ds
  max_connections: %d

database:
  driver: postgresql
  host: db-%s.internal
  port: 5432
  name: %s_db
  pool_size: %d
  max_idle: %d
  ssl_mode: require

cache:
  driver: redis
  host: redis-%s.internal
  port: 6379
  ttl: %ds
  max_memory: "%dMB"

logging:
  level: info
  format: json
  output: stdout
  sentry_dsn: "https://key@sentry.io/%d"

monitoring:
  enabled: true
  metrics_port: 9090
  health_check_interval: %ds
  prometheus:
    namespace: %s
    subsystem: http

`, svc, env,
			rng.Intn(5), rng.Intn(20), rng.Intn(100),
			rng.Intn(10)+1,
			8000+rng.Intn(1000),
			rng.Intn(30)+5, rng.Intn(30)+5,
			rng.Intn(10000)+100,
			svc, svc,
			rng.Intn(50)+5, rng.Intn(20)+2,
			svc,
			rng.Intn(3600)+60,
			rng.Intn(512)+64,
			rng.Intn(1000),
			rng.Intn(30)+5,
			svc,
		))
	}

	data := buf.Bytes()
	if len(data) > target {
		data = data[:target]
	}
	return os.WriteFile(path, data, 0644)
}

// generateBinaryHeaders creates binary data with ELF-like headers and padding.
func generateBinaryHeaders(path string, sizeMB int) error {
	rng := rand.New(rand.NewSource(1337))
	target := sizeMB * 1024 * 1024
	var buf bytes.Buffer

	// ELF magic header pattern
	elfHeader := []byte{0x7F, 0x45, 0x4C, 0x46, 0x02, 0x01, 0x01, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}

	for buf.Len() < target {
		choice := rng.Intn(4)
		switch choice {
		case 0:
			// ELF header block
			buf.Write(elfHeader)
			padding := make([]byte, 48)
			buf.Write(padding)
		case 1:
			// Zero padding (common in binaries)
			zeros := make([]byte, 128+rng.Intn(384))
			buf.Write(zeros)
		case 2:
			// String table (repeated strings)
			strings := []string{".text\x00", ".data\x00", ".bss\x00", ".rodata\x00",
				".symtab\x00", ".strtab\x00", ".shstrtab\x00", ".dynamic\x00"}
			for _, s := range strings {
				buf.WriteString(s)
			}
		case 3:
			// Some semi-structured binary data
			block := make([]byte, 64)
			for i := range block {
				block[i] = byte(i * 3)
			}
			// Add small mutations
			for j := 0; j < 5; j++ {
				block[rng.Intn(64)] = byte(rng.Intn(256))
			}
			buf.Write(block)
		}
	}

	data := buf.Bytes()
	if len(data) > target {
		data = data[:target]
	}
	return os.WriteFile(path, data, 0644)
}

// generatePolynomial creates data following ax²+bx+c mod 256 (best case for fractal engine).
func generatePolynomial(path string, sizeMB int) error {
	target := sizeMB * 1024 * 1024
	data := make([]byte, target)
	// Simple polynomial: data[i] = (i + 5) mod 256
	for i := range data {
		data[i] = byte(i + 5)
	}
	return os.WriteFile(path, data, 0644)
}

// generateHighEntropy creates pseudorandom data (worst case).
func generateHighEntropy(path string, sizeMB int) error {
	rng := rand.New(rand.NewSource(31337))
	target := sizeMB * 1024 * 1024
	data := make([]byte, target)
	for i := range data {
		data[i] = byte(rng.Intn(256))
	}
	return os.WriteFile(path, data, 0644)
}

// generateRealGoRepo concatenates all .go files from the crompressor repo.
func generateRealGoRepo(path string, _ int) error {
	var buf bytes.Buffer
	root := "."

	err := filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() && (d.Name() == ".git" || d.Name() == "benchmark" || d.Name() == "node_modules") {
			return filepath.SkipDir
		}
		if filepath.Ext(p) == ".go" {
			data, err := os.ReadFile(p)
			if err == nil {
				buf.WriteString(fmt.Sprintf("// === FILE: %s ===\n", p))
				buf.Write(data)
				buf.WriteString("\n\n")
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	if buf.Len() == 0 {
		return fmt.Errorf("no .go files found")
	}

	return os.WriteFile(path, buf.Bytes(), 0644)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
