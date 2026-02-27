BINARY := dist/horde
VERSION ?= dev
GOFLAGS := -ldflags="-s -w -X github.com/codebeauty/horde/internal/cli.version=$(VERSION)" -trimpath

build:
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build $(GOFLAGS) -o $(BINARY) ./cmd/horde

install: build
	cp $(BINARY) /usr/local/bin/horde

test:
	go test ./... -v -race

clean:
	rm -rf dist/

.PHONY: build install test clean
