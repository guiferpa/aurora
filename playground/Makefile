install: build assets

assets:
	cp ./src/index.html ./dist/index.html
	cp ./src/main.js ./dist/main.js
	cp $(shell go env GOROOT)/misc/wasm/wasm_exec.js ./dist/wasm_exec.js

build:
	rm -rf ./dist/
	GOOS=js GOARCH=wasm go build -o ./dist/main.wasm ../cmd/playground/main.go
