OUT := api-server
VERSION ?= $(shell git describe --tags --abbrev=0 2>/dev/null || echo dev)

server:
	@echo "Building with version: ${VERSION}"
	CGO_ENABLED=0 go build -v -o ${OUT} -ldflags="-X main.version=${VERSION} -extldflags=-static" ./cmd/server
