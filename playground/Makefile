GOROOT        := $(shell go env GOROOT)
WASM_DIR      := $(GOROOT)/lib/wasm
WASM_JS       := $(WASM_DIR)/wasm_exec.js
GO_VERSION    := $(shell go version | awk '{print $$3}' | sed 's/go//')
WASM_URL      := https://raw.githubusercontent.com/golang/go/master/misc/wasm/wasm_exec.js

# Version injected into the wasm binary (same as CLI via ldflags). Default for local builds.
VERSION       ?= dev
LDFLAGS       := -ldflags "-X github.com/guiferpa/aurora/version.VERSION=$(VERSION)"

install: build assets

# --------------------------------------------------------------------

assets:
	@mkdir -p ./dist
	@cp ./src/index.html ./dist/index.html
	@cp ./src/*.js ./dist/
	@cp $(WASM_JS) ./dist/wasm_exec.js

# --------------------------------------------------------------------

build:
	@rm -rf ./dist/
	@mkdir -p ./dist
	@GOOS=js GOARCH=wasm go build $(LDFLAGS) -o ./dist/main.wasm ../cmd/playground/*.go

# --------------------------------------------------------------------

.PHONY: install build assets
