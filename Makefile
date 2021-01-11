.PHONY: build setup test dist docker-build docker-push \
				run-testnet run-testnet-offline run-mainnet run-mainnet-offline \
				check-testnet-data check-testnet-construction check-mainnet-data

PROJECT             ?= avalanche-rosetta
GIT_COMMIT          ?= $(shell git rev-parse HEAD)
GO_VERSION          ?= $(shell go version | awk {'print $$3'})
WORKDIR             ?= $(shell pwd)
DOCKER_ORG          ?= figmentnetworks
DOCKER_IMAGE        ?= ${DOCKER_ORG}/${PROJECT}
DOCKER_LABEL        ?= latest
DOCKER_TAG          ?= ${DOCKER_IMAGE}:${DOCKER_LABEL}
AVALANCHE_VERSION   ?= v1.1.0
ROSETTA_CLI_VERSION ?= 0.6.6

build:
	go build -o ./rosetta-server ./cmd/server
	go build -o ./rosetta-runner ./cmd/runner

setup:
	go mod download

test:
	go test -v -cover -race ./...

dist:
	@mkdir -p ./bin
	GOOS=linux GOARCH=amd64 go build -o ./bin/avalanche-rosetta_linux-amd64 ./cmd/server
	GOOS=darwin GOARCH=amd64 go build -o ./bin/avalanche-rosetta_darwin-amd64 ./cmd/server

docker-build:
	docker build \
		--build-arg AVALANCHE_VERSION=${AVALANCHE_VERSION} \
		--build-arg ROSETTA_VERSION=${GIT_COMMIT} \
		--build-arg ROSETTA_CLI_VERSION=${ROSETTA_CLI_VERSION} \
		-t ${DOCKER_TAG} \
		-f Dockerfile \
		.

docker-build-standalone:
	docker build \
		--no-cache \
		--build-arg ROSETTA_VERSION=${GIT_COMMIT} \
		-t ${DOCKER_ORG}/${PROJECT}-server:${DOCKER_LABEL} \
		-f Dockerfile.rosetta \
		.

docker-push:
	docker push ${DOCKER_TAG}

# Start the Testnet in ONLINE mode
run-testnet:
	docker run \
		-d \
		-v ${WORKDIR}/.avalanchego:/root/.avalanchego \
		-e AVALANCHE_NETWORK=Fuji \
		-e AVALANCHE_CHAIN=43113 \
		-e AVALANCHE_MODE=online \
		-e ROSETTA_PREFUNDED_ACCOUNT_KEY \
		-e ROSETTA_PREFUNDED_ACCOUNT_ADDRESS \
		--name avalanche-testnet \
		-p 8080:8080 \
		-p 9650:9650 \
		-it \
		${DOCKER_TAG}

# Start the Testnet in OFFLINE mode
run-testnet-offline:
	docker run \
		-d \
		-e AVALANCHE_NETWORK=Fuji \
		-e AVALANCHE_CHAIN=43113 \
		-e AVALANCHE_MODE=offline \
		-e ROSETTA_PREFUNDED_ACCOUNT_KEY \
		-e ROSETTA_PREFUNDED_ACCOUNT_ADDRESS \
		--name avalanche-testnet-offline \
		-p 8080:8080 \
		-p 9650:9650 \
		-it \
		${DOCKER_TAG}

# Start the Mainnet in ONLINE mode
run-mainnet:
	docker run \
		-d \
		-v ${WORKDIR}/.avalanchego:/root/.avalanchego \
		-e AVALANCHE_NETWORK=Mainnet \
		-e AVALANCHE_CHAIN=43114 \
		-e AVALANCHE_MODE=online \
		-e ROSETTA_PREFUNDED_ACCOUNT_KEY \
		-e ROSETTA_PREFUNDED_ACCOUNT_ADDRESS \
		--name avalanche-mainnet \
		-p 8080:8080 \
		-p 9650:9650 \
		-it \
		${DOCKER_TAG}

# Start the Mainnet in ONLINE mode
run-mainnet-offline:
	docker run \
		-d \
		-e AVALANCHE_NETWORK=Mainnet \
		-e AVALANCHE_CHAIN=43114 \
		-e AVALANCHE_MODE=offline \
		-e ROSETTA_PREFUNDED_ACCOUNT_KEY \
		-e ROSETTA_PREFUNDED_ACCOUNT_ADDRESS \
		--name avalanche-mainnet-offline \
		-p 8080:8080 \
		-p 9650:9650 \
		-it \
		${DOCKER_TAG}

# Perform the Testnet data check
check-testnet-data:
	docker exec -it avalanche-testnet /app/rosetta-cli check:data --configuration-file=/app/rosetta-cli-conf/testnet/config.json

# Perform the Testnet construction check
check-testnet-construction:
	docker exec -it avalanche-testnet /app/rosetta-cli check:construction --configuration-file=/app/rosetta-cli-conf/testnet/config.json

# Perform the Mainnet data check
check-mainnet-data:
	docker exec -it avalanche-mainnet /app/rosetta-cli check:data --configuration-file=/app/rosetta-cli-conf/mainnet/config.json