.PHONY: mocks build setup test docker-build \
				run-testnet run-testnet-offline run-mainnet run-mainnet-offline \
				check-testnet-data check-testnet-construction check-testnet-construction-erc20 \
				check-mainnet-data check-mainnet-construction

PROJECT             ?= camino-rosetta
GIT_COMMIT          ?= $(shell git rev-parse HEAD)
GO_VERSION          ?= $(shell go version | awk {'print $$3'})
WORKDIR             ?= $(shell pwd)
DOCKER_ORG          ?= c4tplatform
DOCKER_IMAGE        ?= ${DOCKER_ORG}/${PROJECT}
DOCKER_LABEL        ?= latest
DOCKER_TAG          ?= ${DOCKER_IMAGE}:${DOCKER_LABEL}
CAMINO_VERSION   	?= v0.2.0

build:
	export CGO_CFLAGS="-O -D__BLST_PORTABLE__" && go build -o ./rosetta-server ./cmd/server
	export CGO_CFLAGS="-O -D__BLST_PORTABLE__" && go build -o ./rosetta-runner ./cmd/runner

setup:
	go mod download

test:
	export CGO_CFLAGS="-O -D__BLST_PORTABLE__" && go test -v -cover -race ./...

docker-build:
	docker build \
		--build-arg CAMINO_VERSION=${CAMINO_VERSION} \
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
		-e CAMINO_NETWORK=Columbus \
		-e CAMINO_CHAIN=501 \
		-e CAMINO_MODE=online \
		--name camino-testnet \
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
		-e CAMINO_NETWORK=Columbus \
		-e CAMINO_CHAIN=501 \
		-e CAMINO_MODE=offline \
		--name camino-testnet-offline \
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
		-e CAMINO_NETWORK=Camino \
		-e CAMINO_CHAIN=500 \
		-e CAMINO_MODE=online \
		--name camino-mainnet \
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
		-e CAMINO_NETWORK=Camino \
		-e CAMINO_CHAIN=500 \
		-e CAMINO_MODE=offline \
		--name camino-mainnet-offline \
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

# Perform the Testnet construction check for ERC-20s
check-testnet-construction-erc20:
	rosetta-cli check:construction --configuration-file=rosetta-cli-conf/testnet/config_erc20.json

# Perform the Mainnet data check
check-mainnet-data:
	rosetta-cli check:data --configuration-file=rosetta-cli-conf/mainnet/config.json

# Perform the Mainnet construction check
check-mainnet-construction:
	rosetta-cli check:construction --configuration-file=rosetta-cli-conf/mainnet/config.json

mocks:
	rm -rf mocks;
	mockery --dir client --all --case underscore --outpkg client --output mocks/client;
