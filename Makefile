BINARY := dist/panel
VERSION ?= dev
GOFLAGS := -ldflags="-s -w -X github.com/codebeauty/panel/internal/cli.version=$(VERSION)" -trimpath

build:
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build $(GOFLAGS) -o $(BINARY) ./cmd/panel

install: build
	cp $(BINARY) /usr/local/bin/panel

test:
	go test ./... -v -race

clean:
	rm -rf dist/

.PHONY: build install test clean
