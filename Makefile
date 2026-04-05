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

clean:
	rm -rf bin/ .go-tmp/

lint:
	go vet ./...
