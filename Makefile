BINARY       := catchpoint-exporter
MAIN         := ./cmd/catchpoint-exporter

GO     := go
GOTEST := $(GO) test

GOPATH := $(shell $(GO) env GOPATH)
pkgs   := ./...

GOVULNCHECK_VERSION ?= 617f44b718537dccdea1915395650e0529e3b72e # v1.7.0
GOVULNCHECK          = $(GOPATH)/bin/govulncheck

GOLANGCI_LINT_VERSION ?= c0d3ddc9cf3faa61a4e378e879ece580256d76e5 # v2.12.2
GOLANGCI_LINT          = $(GOPATH)/bin/golangci-lint

.PHONY: all build test lint fmt vet golangci-lint vuln-check gosec-check security-check clean

all: test build security-check

build:
	$(GO) build  -o $(BINARY) $(MAIN)

test:
	$(GOTEST) ./... -v

lint: fmt vet golangci-lint

golangci-lint:
	@command -v $(GOLANGCI_LINT) >/dev/null 2>&1 || { echo "golangci-lint not installed. Install: go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)"; exit 1; }
	$(GOLANGCI_LINT) run ./...

fmt:
	$(GO) fmt ./...

vet:
	$(GO) vet ./...

vuln-check:
	@echo ">> Running govulncheck..."
	@command -v $(GOVULNCHECK) >/dev/null 2>&1 || { echo "govulncheck not installed. Install: go install golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION)"; exit 1; }
	$(GOVULNCHECK) ./...
	@echo ">> govulncheck passed!"

gosec-check:
	@echo ">> Running gosec via golangci-lint..."
	@command -v $(GOLANGCI_LINT) >/dev/null 2>&1 || { echo "golangci-lint not installed. Install: go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)"; exit 1; }
	$(GOLANGCI_LINT) run --enable-only gosec $(pkgs)
	@echo ">> Security checks passed!"

security-check: vuln-check gosec-check

clean:
	rm -f $(BINARY)
