# ------------------------------------------------------------------------------
# Build avalanche
# ------------------------------------------------------------------------------
FROM golang:1.15 AS avalanche

ARG AVALANCHE_VERSION

RUN git clone https://github.com/ava-labs/avalanchego.git \
  /go/src/github.com/ava-labs/avalanchego

WORKDIR /go/src/github.com/ava-labs/avalanchego

RUN git checkout $AVALANCHE_VERSION && \
    ./scripts/build.sh

# ------------------------------------------------------------------------------
# Build avalanche rosetta
# ------------------------------------------------------------------------------
FROM golang:1.15 AS rosetta

ARG ROSETTA_VERSION

RUN git clone https://github.com/figment-networks/avalanche-rosetta.git \
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

# Install dependencies
RUN apt-get update -y && \
    apt-get install -y wget

WORKDIR /app

# Install avalanche binaries
COPY --from=avalanche \
  /go/src/github.com/ava-labs/avalanchego/build/avalanchego \
  /app/avalanchego

# Install plugins
COPY --from=avalanche \
  /go/src/github.com/ava-labs/avalanchego/build/plugins/* \
  /app/plugins/

# Install rosetta server
COPY --from=rosetta \
  /go/src/github.com/figment-networks/avalanche-rosetta/avalanche-rosetta \
  /app/rosetta

# Install service start script
COPY --from=rosetta \
  /go/src/github.com/figment-networks/avalanche-rosetta/docker/start.sh \
  /app/start

# Install rosetta CLI
RUN wget https://github.com/coinbase/rosetta-cli/releases/download/v0.6.2/rosetta-cli-0.6.2-linux-amd64.tar.gz && \
    tar -xzf rosetta-cli-0.6.2-linux-amd64.tar.gz && \
    mv rosetta-cli-0.6.2-linux-amd64 rosetta-cli && \
    rm *.tar.gz

EXPOSE 9650
EXPOSE 8081

CMD ["/app/start"]
