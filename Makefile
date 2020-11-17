.PHONY: build test dist docker-build docker-push

PROJECT      ?= avalanche-rosetta
GIT_COMMIT   ?= $(shell git rev-parse HEAD)
GO_VERSION   ?= $(shell go version | awk {'print $$3'})
DOCKER_IMAGE ?= figmentnetworks/${PROJECT}
DOCKER_TAG   ?= latest

build:
	go build -o ./avalanche-rosetta ./cmd/server

test:
	go test -v -cover -race ./...

dist:
	@mkdir -p ./bin
	GOOS=linux GOARCH=amd64 go build -o ./bin/avalanche-rosetta_linux-amd64 ./cmd/server
	GOOS=darwin GOARCH=amd64 go build -o ./bin/avalanche-rosetta_darwin-amd64 ./cmd/server

docker-build:
	docker build -t ${DOCKER_IMAGE}:${DOCKER_TAG} -f Dockerfile .

docker-push:
	docker push ${DOCKER_IMAGE}:${DOCKER_TAG}
