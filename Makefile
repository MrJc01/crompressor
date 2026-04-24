.PHONY: build test bench clean lint

build:
	@mkdir -p bin
	go build -o bin/crompressor ./cmd/crompressor

test:
	@mkdir -p .go-tmp
	GOTMPDIR="$(shell pwd)/.go-tmp" go test -v -race ./...

bench:
	@mkdir -p .go-tmp
	GOTMPDIR="$(shell pwd)/.go-tmp" go test -bench=. -benchmem -benchtime=10s ./...

lint:
	go vet ./...

clean:
	rm -rf bin/ .go-tmp/

# --- Development helpers ---

gen-codebook:
	@mkdir -p testdata
	go run scripts/gen_mini_codebook.go

demo: build gen-codebook
	@echo "=== Generating test sample (1MB) ==="
	@mkdir -p testdata
	@head -c 1048576 /dev/urandom > testdata/sample_1mb.bin
	@echo "=== Compressing ==="
	./bin/crompressor pack --input testdata/sample_1mb.bin --output /tmp/test.crom --codebook testdata/mini.cromdb
	@echo "=== Decompressing ==="
	./bin/crompressor unpack --input /tmp/test.crom --output /tmp/restored.bin --codebook testdata/mini.cromdb
	@echo "=== Verifying integrity ==="
	./bin/crompressor verify --original testdata/sample_1mb.bin --restored /tmp/restored.bin
