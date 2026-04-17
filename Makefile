BINARY       := catchpoint-exporter
MODULE       := github.com/grafana/catchpoint-prometheus-exporter
MAIN         := ./cmd/catchpoint-exporter
DOCKER_IMAGE := catchpoint-prometheus-exporter

GO     := go
GOTEST := $(GO) test

VERSION    ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT     ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")


.PHONY: all build test lint fmt vet golangci-lint clean

all: test build

build:
	$(GO) build  -o $(BINARY) $(MAIN)

test:
	$(GOTEST) ./...

lint: fmt vet golangci-lint

golangci-lint:
	golangci-lint run ./...

fmt:
	$(GO) fmt ./...

vet:
	$(GO) vet ./...

clean:
	rm -f $(BINARY)
