# ------------------------------------------------------------------------------
# Build avalanche
# ------------------------------------------------------------------------------
FROM golang:1.16 AS avalanche

ARG AVALANCHE_VERSION

RUN git clone https://github.com/ava-labs/avalanchego.git \
  /go/src/github.com/ava-labs/avalanchego

WORKDIR /go/src/github.com/ava-labs/avalanchego

RUN git checkout $AVALANCHE_VERSION && \
    ./scripts/build.sh

# ------------------------------------------------------------------------------
# Build avalanche rosetta
# ------------------------------------------------------------------------------
FROM golang:1.16 AS rosetta

ARG ROSETTA_VERSION

# RUN git clone https://github.com/figment-networks/avalanche-rosetta.git \
COPY . \
  /go/src/github.com/figment-networks/avalanche-rosetta

WORKDIR /go/src/github.com/figment-networks/avalanche-rosetta

ENV CGO_ENABLED=1
ENV GOARCH=amd64
ENV GOOS=linux

RUN git checkout $ROSETTA_VERSION && \
    go mod download

RUN \
  GO_VERSION=$(go version | awk {'print $3'}) \
  GIT_COMMIT=$(git rev-parse HEAD) \
  make build

# ------------------------------------------------------------------------------
# Target container for running the node and rosetta server
# ------------------------------------------------------------------------------
FROM ubuntu:18.04

ARG ROSETTA_CLI_VERSION

# Install dependencies
RUN apt-get update -y && \
    apt-get install -y wget

WORKDIR /app

# Install avalanche daemon
COPY --from=avalanche \
  /go/src/github.com/ava-labs/avalanchego/build/avalanchego \
  /app/avalanchego

# Install pre-upgrade binaries
COPY --from=avalanche \
  /go/src/github.com/ava-labs/avalanchego/build/avalanchego-preupgrade/avalanchego-process \
  /app/avalanchego-preupgrade/avalanchego-process
COPY --from=avalanche \
  /go/src/github.com/ava-labs/avalanchego/build/avalanchego-preupgrade/plugins/evm \
  /app/avalanchego-preupgrade/plugins/evm

# Install latest binaries
COPY --from=avalanche \
  /go/src/github.com/ava-labs/avalanchego/build/avalanchego-latest/avalanchego-process \
  /app/avalanchego-latest/avalanchego-process
COPY --from=avalanche \
  /go/src/github.com/ava-labs/avalanchego/build/avalanchego-latest/plugins/evm \
  /app/avalanchego-latest/plugins/evm

# Install rosetta server
COPY --from=rosetta \
  /go/src/github.com/figment-networks/avalanche-rosetta/rosetta-server \
  /app/rosetta-server

# Install rosetta runner
COPY --from=rosetta \
  /go/src/github.com/figment-networks/avalanche-rosetta/rosetta-runner \
  /app/rosetta-runner

# Install service start script
COPY --from=rosetta \
  /go/src/github.com/figment-networks/avalanche-rosetta/docker/entrypoint.sh \
  /app/entrypoint.sh

EXPOSE 9650
EXPOSE 8080

ENTRYPOINT ["/app/entrypoint.sh"]
