BINARY       := catchpoint-exporter
MAIN         := ./cmd/catchpoint-exporter

GO     := go
GOTEST := $(GO) test

GOPATH := $(shell $(GO) env GOPATH)
pkgs   := ./...

GOVULNCHECK_VERSION ?= 0782b76014f15f24e22a438f30f308df42899ba1 # v1.3.0
GOVULNCHECK          = $(GOPATH)/bin/govulncheck

.PHONY: all build test lint fmt vet golangci-lint vuln-check gosec-check security-check clean

all: test build security-check

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

vuln-check:
	@echo ">> Running govulncheck..."
	@command -v $(GOVULNCHECK) >/dev/null 2>&1 || { echo "govulncheck not installed. Install: go install golang.org/x/vuln/cmd/govulncheck@0782b76014f15f24e22a438f30f308df42899ba1 # v1.3.0"; exit 1; }
	$(GOVULNCHECK) ./...
	@echo ">> govulncheck passed!"

gosec-check:
	@echo ">> Running gosec via golangci-lint..."
	@command -v golangci-lint >/dev/null 2>&1 || { echo "golangci-lint not installed. Install: https://golangci-lint.run/docs/welcome/install/"; exit 1; }
	golangci-lint run --enable-only gosec $(pkgs)
	@echo ">> Security checks passed!"

security-check: vuln-check gosec-check

clean:
	rm -f $(BINARY)
