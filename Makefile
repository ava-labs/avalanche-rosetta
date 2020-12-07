.PHONY: build test dist docker-build docker-push run-testnet run-testnet-offline run-mainnet run-mainnet-offline

PROJECT           ?= avalanche-rosetta
GIT_COMMIT        ?= $(shell git rev-parse HEAD)
GO_VERSION        ?= $(shell go version | awk {'print $$3'})
DOCKER_ORG        ?= figmentnetworks
DOCKER_IMAGE      ?= ${DOCKER_ORG}/${PROJECT}
DOCKER_LABEL      ?= latest
DOCKER_TAG        ?= ${DOCKER_IMAGE}:${DOCKER_LABEL}
AVALANCHE_VERSION ?= v1.1.0

build:
	go build -o ./avalanche-rosetta ./cmd/server

test:
	go test -v -cover -race ./...

dist:
	@mkdir -p ./bin
	GOOS=linux GOARCH=amd64 go build -o ./bin/avalanche-rosetta_linux-amd64 ./cmd/server
	GOOS=darwin GOARCH=amd64 go build -o ./bin/avalanche-rosetta_darwin-amd64 ./cmd/server

docker-build:
	docker build \
		--no-cache \
		--build-arg AVALANCHE_VERSION=${AVALANCHE_VERSION} \
		--build-arg ROSETTA_VERSION=${GIT_COMMIT} \
		-t ${DOCKER_TAG} \
		-f Dockerfile \
		.

docker-push:
	docker push ${DOCKER_TAG}

# Start the Testnet in ONLINE mode
run-testnet:
	docker run \
		-e AVALANCHE_NETWORK=Fuji \
		-e AVALANCHE_CHAIN=43113 \
		-e AVALANCHE_MODE=online \
		--rm -p 8080:8080 -p 9650:9650 -it ${DOCKER_TAG}

# Start the Testnet in OFFLINE mode
run-testnet-offline:
	docker run \
		-e AVALANCHE_NETWORK=Fuji \
		-e AVALANCHE_CHAIN=43113 \
		-e AVALANCHE_MODE=offline \
		--rm -p 8080:8080 -p 9650:9650 -it ${DOCKER_TAG}

# Start the Mainnet in ONLINE mode
run-mainnet:
	docker run \
		-e AVALANCHE_NETWORK=Mainnet \
		-e AVALANCHE_CHAIN=43114 \
		-e AVALANCHE_MODE=online \
		--rm -p 8080:8080 -p 9650:9650 -it ${DOCKER_TAG}

# Start the Mainnet in ONLINE mode
run-mainnet-offline:
	docker run \
		-e AVALANCHE_NETWORK=Mainnet \
		-e AVALANCHE_CHAIN=43114 \
		-e AVALANCHE_MODE=offline \
		--rm -p 8080:8080 -p 9650:9650 -it ${DOCKER_TAG}
