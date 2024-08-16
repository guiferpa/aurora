install: test build

build:
	@CGO_ENABLED=1 go build -race -o ./target/bin/aurora ./cmd/aurora/*.go

test:
	@go test ./... -test.v -cover

