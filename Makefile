
# Default target
all: build

BUILD_TIME=`date +%FT%T%z`
GIT_REV=`git rev-parse --short HEAD`
GO_VERSION=$(shell go version)
GIT_VERSION=$(shell git describe --tags --exact-match 2>/dev/null || git rev-parse --short HEAD)
VERSION?=${GIT_VERSION}
LDFLAGS=-ldflags "-w -s \
-X 'github.com/wayjam/tv-mixproxy/internal.Version=${VERSION}' \
-X 'github.com/wayjam/tv-mixproxy/internal.GitRev=${GIT_REV}' \
-X 'github.com/wayjam/tv-mixproxy/internal.BuildTime=${BUILD_TIME}' \
-X 'github.com/wayjam/tv-mixproxy/internal.GoVersion=${GO_VERSION}' \
"

# Build the Go binary
.PHONY: build
build:
	go build ${LDFLAGS} -o build/tv-mixproxy ./cmd/tv-mixproxy

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

