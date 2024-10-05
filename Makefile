
# Default target
all: build

# Build the Go binary
.PHONY: build
build:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o build/tvbox-mixproxy ./cmd/tvbox-mixproxy

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
	docker build -t ghcr.io/tvbox-mixproxy/tvbox-mixproxy:latest -f Dockerfile .

# Clean up
.PHONY: clean
clean:
	rm -f ./build/*

