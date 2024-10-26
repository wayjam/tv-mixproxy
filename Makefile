
# Default target
all: build

# Build the Go binary
.PHONY: build
build:
	go build -o build/tv-mixproxy ./cmd/tv-mixproxy

.PHONY: vet
vet:
	go vet ./...; true

# Run tests
.PHONY: test
test:
	go test ./...

# Build Docker image
.PHONY: image
image:
	docker build -t ghcr.io/tv-mixproxy/tv-mixproxy:latest -f Dockerfile .

# Clean up
.PHONY: clean
clean:
	rm -f ./build/*

