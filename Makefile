BINARY       := catchpoint-exporter
MAIN         := ./cmd/catchpoint-exporter

GO     := go
GOTEST := $(GO) test



.PHONY: all build test lint fmt vet golangci-lint clean

all: test build

build:
	$(GO) build  -o $(BINARY) $(MAIN)

test:
	$(GOTEST) ./... -v

lint: fmt vet golangci-lint

golangci-lint:
	golangci-lint run ./...

fmt:
	$(GO) fmt ./...

vet:
	$(GO) vet ./...

clean:
	rm -f $(BINARY)
