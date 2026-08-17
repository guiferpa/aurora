SHELL=/bin/sh
GOBIN ?= $(shell go env GOBIN)
# `go env GOBIN` is empty unless someone set it, and `go install` then falls back to
# GOPATH/bin. Without this the tool paths below point at /tparse and /golangci-lint, which
# exist nowhere — it works on a machine with GOBIN set and fails everywhere else.
ifeq ($(GOBIN),)
GOBIN := $(shell go env GOPATH)/bin
endif
BIN ?= ./target/bin
PKGS = $(shell go list ./... | grep -v examples)
# Every Go file plus the module files: prerequisites of the binary, so editing source
# rebuilds it instead of leaving a stale binary behind.
SOURCES = $(shell find . -type f -name '*.go' -not -path './.git/*') go.mod go.sum
LINTER = $(GOBIN)/golangci-lint
ACT_BIN = $(GOBIN)/act
TPARSE_BIN = $(GOBIN)/tparse
GODEPGRAPH_BIN = $(GOBIN)/godepgraph


# Execute all meaningful jobs from Makefile to release the project's binary
all: test lint build-force

# Everything a change has to survive before it is committed: it compiles for both targets,
# the tests pass and the linter is quiet.
#
# The wasm build is in here because the playground is the one target that does not come out
# of `go build ./...`, and it is the one that breaks silently — nothing else compiles for it.
check: build wasm test lint

# Compile every package, without producing binaries.
build:
	@go build ./...

# Compile for the browser, which is what the playground runs on.
wasm:
	@GOOS=js GOARCH=wasm go build ./...

build-force: clean aurora aurorals

aurora: $(BIN)/aurora

$(BIN)/aurora: $(SOURCES)
	@mkdir -p $(BIN)
	@CGO_ENABLED=0 go build -race -o $(BIN)/aurora ./cmd/aurora/*.go

aurorals: $(BIN)/aurorals

$(BIN)/aurorals: $(SOURCES)
	@mkdir -p $(BIN)
	@CGO_ENABLED=0 go build -race -o $(BIN)/aurorals ./cmd/aurorals

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

# Report how complex the code is, without failing: it is the instrument, not the gate.
#
# Cognitive complexity is the number that matters and the one `make lint` enforces — it
# measures how hard a function is to follow. Cyclomatic complexity comes along as
# information only: it counts branches, so it punishes a flat lookup switch that anyone
# reads in seconds (ResolveOpCode scores 105 and is trivial) while missing deep nesting.
complexity: $(LINTER)
	@$(LINTER) run ./... --timeout 10m --no-config --default=none \
		--enable=gocognit --enable=gocyclo --enable=funlen --enable=nestif || true

# This jobs is to simulate github ci environment for tests github action workflows
act: $(ACT_BIN)
	$(ACT_BIN) --container-architecture linux/amd64 --platform ubuntu-latest=node:buster --rm

$(ACT_BIN):
	@echo "==> Installing act..."
	@curl -sSfL https://raw.githubusercontent.com/nektos/act/38e43bd51f66493057857f6d743153c874a7178f/install.sh | sh -s -- -b ${GOBIN}

# It's a great job to take a look to source code coverage using a friendly view
cover-html: test
	@go tool cover -html=coverage.out

# Generate dependency graph image using godepgraph and Graphviz (dot).
# Prerequisites:
#   - macOS: brew install graphviz
#   - Debian/Ubuntu: sudo apt-get install graphviz
#   - Arch Linux: sudo pacman -S graphviz
#   - Fedora/RHEL: sudo dnf install graphviz
depgraph: $(GODEPGRAPH_BIN)
	@echo "==> Generating dependency graph image..."
	@$(GODEPGRAPH_BIN) -novendor -nostdlib -s ./cmd/aurora | dot -Tsvg -o graph.svg
	@echo "==> graph.png successfully generated."

$(TPARSE_BIN):
	@echo "==> Installing tparse..."
	@go install github.com/mfridman/tparse@latest

$(GODEPGRAPH_BIN):
	@echo "==> Installing godepgraph..."
	@go install github.com/kisielk/godepgraph@latest

.PHONY: all check build wasm build-force aurora aurorals test bench lint complexity act cover-html clean depgraph
