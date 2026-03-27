.PHONY: build test bench clean lint gen-codebook demo stress searchbench gen-realworld train-standard demo-real stress-vfs

build:
	@mkdir -p bin
	go build -o bin/crom ./cmd/crom

test:
	go test -v -race ./...

bench:
	go test -bench=. -benchmem -benchtime=10s ./...

clean:
	rm -rf bin/

lint:
	go vet ./...

gen-codebook:
	@mkdir -p testdata
	go run scripts/gen_mini_codebook.go

demo: build gen-codebook
	@echo "=== Gerando arquivo de amostra (1MB) ==="
	@mkdir -p testdata
	@head -c 1048576 /dev/urandom > testdata/sample_1mb.bin
	@echo "=== Compilando arquivo de teste ==="
	./bin/crom pack --input testdata/sample_1mb.bin --output /tmp/test.crom --codebook testdata/mini.cromdb
	@echo "=== Descompilando ==="
	./bin/crom unpack --input /tmp/test.crom --output /tmp/restored.bin --codebook testdata/mini.cromdb
	@echo "=== Verificando integridade ==="
	sha256sum testdata/sample_1mb.bin /tmp/restored.bin
	./bin/crom verify --original testdata/sample_1mb.bin --restored /tmp/restored.bin

stress: build gen-codebook
	@echo "=== Gerando arquivo de stress (100MB) ==="
	@mkdir -p testdata
	@head -c 104857600 /dev/urandom > testdata/sample_100mb.bin
	@echo "=== PACK (100MB) ==="
	./bin/crom pack --input testdata/sample_100mb.bin --output /tmp/stress.crom --codebook testdata/mini.cromdb --concurrency 4
	@echo "=== UNPACK (100MB) ==="
	./bin/crom unpack --input /tmp/stress.crom --output /tmp/stress_restored.bin --codebook testdata/mini.cromdb
	@echo "=== VERIFY (100MB) ==="
	./bin/crom verify --original testdata/sample_100mb.bin --restored /tmp/stress_restored.bin
	@rm -f testdata/sample_100mb.bin /tmp/stress.crom /tmp/stress_restored.bin

searchbench:
	@echo "=== Benchmark A/B: Linear vs LSH ==="
	go test -bench=BenchmarkLinearSearcher -benchmem -benchtime=3s ./internal/search/
	go test -bench=BenchmarkLSHSearcher -benchmem -benchtime=3s ./internal/search/

gen-realworld:
	@echo "=== Gerando Dados Reais (Go, Logs, Config, Binários) ==="
	go run scripts/gen_real_world.go

train-standard: build gen-realworld
	@echo "=== Treinando Codebook a partir do Real-World Data ==="
	./bin/crom train --input testdata/real_world --output testdata/trained.cromdb --size 8192 --concurrency 4

demo-real: build
	@echo "=== Testando Compressão com Codebook Treinado ==="
	@mkdir -p /tmp/crom_demo
	@echo "package example\nfunc Sum(a, b int) int {\n\treturn a + b\n}\nfunc Test() {\n\tfmt.Println(\"Hello, Real World\")\n}\n" > /tmp/crom_demo/source.go
	@cat testdata/real_world/go_source/part_000.dat | head -c 1048576 >> /tmp/crom_demo/source.go
	./bin/crom pack --input /tmp/crom_demo/source.go --output /tmp/crom_demo/source.crom --codebook testdata/trained.cromdb
	./bin/crom unpack --input /tmp/crom_demo/source.crom --output /tmp/crom_demo/source_restored.go --codebook testdata/trained.cromdb
	./bin/crom verify --original /tmp/crom_demo/source.go --restored /tmp/crom_demo/source_restored.go

stress-vfs: build gen-codebook
	@echo "=== Stress Test: Random Access via RandomReader ==="
	go test -v -run TestRandomAccess -count=1 ./internal/vfs/

run-node:
	@echo '--- Iniciando Daemon P2P ---'
	./bin/crom daemon --codebook testdata/trained.cromdb --data-dir ./crom-data --port 4001

