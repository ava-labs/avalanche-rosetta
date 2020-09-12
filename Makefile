.PHONY: build

build:
	go build -o ./avalanche-rosetta ./cmd/server

test:
	go test -cover -race ./...
