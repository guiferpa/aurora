SHELL=/bin/sh
GOBIN ?= $(shell go env GOBIN)
BIN ?= ./target/bin
PKGS = $(shell go list ./... | grep -v examples)
LINTER = $(GOBIN)/golangci-lint
ACT_BIN = $(GOBIN)/act
TPARSE_BIN = $(GOBIN)/tparse


# Execute all meaningful jobs from Makefile to release the project's binary
all: test lint build-force

build-force: clean aurora aurorals

aurora: $(BIN)/aurora

aurorals: $(BIN)/aurorals

$(BIN)/aurora:
	@CGO_ENABLED=0 go build -race -o $(BIN)/aurora ./cmd/aurora/*.go

$(BIN)/aurorals:
	@CGO_ENABLED=0 go build -race -o $(BIN)/aurorals ./cmd/aurorals/*.go

clean:
	@rm -rf $(BIN)

# Run tests (writes coverage.out for make cover-html)
test: $(TPARSE_BIN)
	@go test $(PKGS) -v -json -race -buildvcs -cover -covermode=atomic -coverprofile=coverage.out -test.v | $(TPARSE_BIN) -pass -follow

# Run benchmarks of source code
bench:
	@go test $(PKGS) -v -race -buildvcs -bench=. -benchmem -cpu=1,2,4,12

# Run lint
lint: $(LINTER)
	@$(LINTER) run ./... --timeout 10m

$(LINTER):
	@echo "==> Installing linter..."
	@curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/f7cf900a4f6021580b7b962645872bbd453f11f2/install.sh | sh -s -- -b ${GOBIN} v2.7.2

# This jobs is to simulate github ci environment for tests github action workflows
act: $(ACT_BIN)
	$(ACT_BIN) --container-architecture linux/amd64 --platform ubuntu-latest=node:buster --rm

$(ACT_BIN):
	@echo "==> Installing act..."
	@curl -sSfL https://raw.githubusercontent.com/nektos/act/38e43bd51f66493057857f6d743153c874a7178f/install.sh | sh -s -- -b ${GOBIN}

# It's a great job to take a look to source code coverage using a friendly view
cover-html: test
	@go tool cover -html=coverage.out

$(TPARSE_BIN):
	@echo "==> Installing tparse..."
	@go install github.com/mfridman/tparse@latest

.PHONY: all build build-force aurora aurorals test bench lint act cover-html clean
