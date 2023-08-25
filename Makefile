.PHONY: mocks build setup test docker-build \
				run-testnet run-testnet-offline run-mainnet run-mainnet-offline \
				check-testnet-data check-testnet-construction check-testnet-construction-erc20 \
				check-mainnet-data check-mainnet-construction

PROJECT             ?= avalanche-rosetta
GIT_COMMIT          ?= $(shell git rev-parse HEAD)
GO_VERSION          ?= $(shell go version | awk {'print $$3'})
WORKDIR             ?= $(shell pwd)
DOCKER_ORG          ?= avaplatform
DOCKER_IMAGE        ?= ${DOCKER_ORG}/${PROJECT}
DOCKER_LABEL        ?= latest
DOCKER_TAG          ?= ${DOCKER_IMAGE}:${DOCKER_LABEL}
AVALANCHE_VERSION   ?= v1.10.9

build:
	export CGO_CFLAGS="-O -D__BLST_PORTABLE__" && go build -o ./rosetta-server ./cmd/server
	export CGO_CFLAGS="-O -D__BLST_PORTABLE__" && go build -o ./rosetta-runner ./cmd/runner

setup:
	go mod download

test:
	export CGO_CFLAGS="-O -D__BLST_PORTABLE__" && go test -v -cover -race ./...

docker-build:
	docker build \
		--build-arg AVALANCHE_VERSION=${AVALANCHE_VERSION} \
		--build-arg ROSETTA_VERSION=${GIT_COMMIT} \
		-t ${DOCKER_TAG} \
		-f Dockerfile \
		.

# Start the Testnet in ONLINE mode
run-testnet:
	docker run \
		--rm \
		-d \
		-v ${WORKDIR}/data:/data \
		-e AVALANCHE_NETWORK=Fuji \
		-e AVALANCHE_CHAIN=43113 \
		-e AVALANCHE_MODE=online \
		--name avalanche-testnet \
		-p 8080:8080 \
		-p 9650:9650 \
		-p 9651:9651 \
		-it \
		${DOCKER_TAG}

# Start the Testnet in OFFLINE mode
run-testnet-offline:
	docker run \
		--rm \
		-d \
		-e AVALANCHE_NETWORK=Fuji \
		-e AVALANCHE_CHAIN=43113 \
		-e AVALANCHE_MODE=offline \
		--name avalanche-testnet-offline \
		-p 8080:8080 \
		-p 9650:9650 \
		-it \
		${DOCKER_TAG}

# Start the Mainnet in ONLINE mode
run-mainnet:
	docker run \
		--rm \
		-d \
		-v ${WORKDIR}/data:/data \
		-e AVALANCHE_NETWORK=Mainnet \
		-e AVALANCHE_CHAIN=43114 \
		-e AVALANCHE_MODE=online \
		--name avalanche-mainnet \
		-p 8080:8080 \
		-p 9650:9650 \
		-p 9651:9651 \
		-it \
		${DOCKER_TAG}

# Start the Mainnet in ONLINE mode
run-mainnet-offline:
	docker run \
		--rm \
		-d \
		-e AVALANCHE_NETWORK=Mainnet \
		-e AVALANCHE_CHAIN=43114 \
		-e AVALANCHE_MODE=offline \
		--name avalanche-mainnet-offline \
		-p 8080:8080 \
		-p 9650:9650 \
		-it \
		${DOCKER_TAG}

# Perform the Testnet data check
check-testnet-data:
	rosetta-cli check:data --configuration-file=rosetta-cli-conf/testnet/config.json

# Perform the Testnet construction check
check-testnet-construction:
	rosetta-cli check:construction --configuration-file=rosetta-cli-conf/testnet/config.json

# Perform the Testnet construction check for ERC-20 transfers
check-testnet-construction-erc20:
	rosetta-cli check:construction --configuration-file=rosetta-cli-conf/testnet/config_erc20.json

# Perform the Testnet construction check for unwrap bridge tokens
check-testnet-construction-unwrap:
	rosetta-cli check:construction --configuration-file=rosetta-cli-conf/testnet/config_unwrap.json

# Perform the Mainnet data check
check-mainnet-data:
	rosetta-cli check:data --configuration-file=rosetta-cli-conf/mainnet/config.json

# Perform the Mainnet construction check
check-mainnet-construction:
	rosetta-cli check:construction --configuration-file=rosetta-cli-conf/mainnet/config.json

mocks:
	rm -rf mocks;
	mockery --dir client --all --case underscore --outpkg client --output mocks/client;
	mockery --dir service --name '.*Backend' --case underscore --outpkg chain --output mocks/service;
	mockery --dir service/backend --all --case underscore --outpkg chain --output mocks/service/backend;
	mockery --dir service/backend/pchain/indexer --all --case underscore --outpkg chain --output mocks/service/backend/pchain/indexer;
