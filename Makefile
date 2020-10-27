.PHONY: build test docker-build

PROJECT      ?= avalanche-rosetta
GIT_COMMIT   ?= $(shell git rev-parse HEAD)
GO_VERSION   ?= $(shell go version | awk {'print $$3'})
DOCKER_IMAGE ?= figmentnetworks/${PROJECT}
DOCKER_TAG   ?= latest

build:
	go build -o ./avalanche-rosetta ./cmd/server

setup:
	# noop for now

test:
	go test -cover -race ./...

docker-build:
	docker build -t ${DOCKER_IMAGE}:${DOCKER_TAG} -f Dockerfile .
