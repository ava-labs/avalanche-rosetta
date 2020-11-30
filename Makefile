.PHONY: build test dist docker-build docker-push

PROJECT      ?= avalanche-rosetta
GIT_COMMIT   ?= $(shell git rev-parse HEAD)
GO_VERSION   ?= $(shell go version | awk {'print $$3'})
DOCKER_IMAGE ?= figmentnetworks/${PROJECT}
DOCKER_LABEL ?= latest
DOCKER_TAG   ?= ${DOCKER_IMAGE}:${DOCKER_LABEL}

build:
	go build -o ./avalanche-rosetta ./cmd/server

test:
	go test -v -cover -race ./...

dist:
	@mkdir -p ./bin
	GOOS=linux GOARCH=amd64 go build -o ./bin/avalanche-rosetta_linux-amd64 ./cmd/server
	GOOS=darwin GOARCH=amd64 go build -o ./bin/avalanche-rosetta_darwin-amd64 ./cmd/server

docker-build:
	docker build --no-cache -t ${DOCKER_TAG} -f Dockerfile .

docker-push:
	docker push ${DOCKER_TAG}

run-testnet:
	docker run -e AVALANCHE_NETWORK=testnet -e AVALANCHE_CHAIN=43113 --rm -p 8081:8081 -p 9650:9650 -it ${DOCKER_TAG}

run-mainnet:
	docker run -e AVALANCHE_NETWORK=mainnet -e AVALANCHE_CHAIN=43114 --rm -p 8081:8081 -p 9650:9650 -it ${DOCKER_TAG}
