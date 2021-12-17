<div align="center">
  <img src="resources/AvalancheLogoRed.png?raw=true">
</div>

---

# Avalanche Rosetta

[Rosetta][1] server implementation for [Avalanche][2] C-Chain.

## Requirements

In order to run the Avalanche Rosetta server you will need access to [Avalanche][3]
services via RPC. More info in available APIs found [here][4].

See AvalancheGo documentation on how to run the chain node locally. If you don't run
the Avalanche node yourself you might use the [hosted API provided by Ava Labs][5].

## Installation

Clone repository, then build the rosetta server by running the following commands:

```bash
make setup
make build
```

If successful, you will have `rosetta-server` binary in your current directory.

## Usage

Before you start running the server you need to create a configuration file:

```json
{
  "rpc_endpoint": "https://api.avax-test.network",
  "mode": "online",
  "listen_addr": "0.0.0.0:8080",
  "genesis_block_hash" :"0x31ced5b9beb7f8782b014660da0cb18cc409f121f408186886e1ca3e8eeca96b",
}
```

Start the server by running the following command:

```bash
./rosetta-server -config=./config.json
```
## Configuration

Full configuration example:

```json
{
  "mode": "online",
  "rpc_endpoint": "http://localhost:9650",
  "listen_addr": "0.0.0.0:8080",
  "network_name": "Fuji",
  "chain_id": 43113,
  "log_requests": true,
  "genesis_block_hash" :"0x31ced5b9beb7f8782b014660da0cb18cc409f121f408186886e1ca3e8eeca96b",
}
```

Where:

| Name          | Type    | Default | Description
|---------------|---------|---------|-------------------------------------------
| mode          | string  | `online` | Mode of operations. One of: `online`, `offline`
| rpc_endpoint  | string  | `http://localhost:9650` | Avalanche RPC endpoint
| listen_addr   | string  | `http://localhost:8080` | Rosetta server listen address (host/port)
| network_name  | string  | - | Avalanche network name
| chain_id      | integer | - | Avalanche C-Chain ID
| log_requests  | bool    | `false` | Enable request body logging

### RPC Endpoints

List of all available Rosetta RPC server endpoints

| Method | Path                     | Status | Description
|--------|--------------------------|--------|----------------------------------
| POST   | /network/list            | Y      | Get List of Available Networks
| POST   | /network/status          | Y      | Get Network Status
| POST   | /network/options         | Y      | Get Network Options
| POST   | /block                   | Y      | Get a Block
| POST   | /block/transaction       | Y      | Get a Block Transaction
| POST   | /account/balance         | Y      | Get an Account Balance
| POST   | /mempool                 | Y      | Get All Mempool Transactions counts
| POST   | /mempool/transaction     | N/A    | Get a Mempool Transaction
| POST   | /construction/combine    | Y      | Create Network Transaction from Signatures
| POST   | /construction/derive     | Y      | Derive an AccountIdentifier from a PublicKey
| POST   | /construction/hash       | Y      | Get the Hash of a Signed Transaction
| POST   | /construction/metadata   | Y      | Get Transaction Construction Metadata
| POST   | /construction/parse      | Y      | Parse a Transaction
| POST   | /construction/payloads   | Y      | Generate an Unsigned Transaction and Signing Payloads
| POST   | /construction/preprocess | Y      | Create a Request to Fetch Metadata
| POST   | /construction/submit     | Y      | Submit a Signed Transaction
| POST   | /call                    | Y      | Perform a Blockchain Call

## Development

Available commands:

- `make build`               - Build the development version of the binary
- `make test`                - Run the test suite
- `make dist`                - Build distribution binaries
- `make docker-build`        - Build a Docker image
- `make docker-push`         - Push a Docker image to the registry
- `make run-testnet`         - Run node and rosetta testnet server
- `make run-testnet-offline` - Run node and rosetta testnet server
- `make run-mainnet`         - Run node and rosetta mainnet server
- `make run-mainnet-offline` - Run node and rosetta mainnet server

## Testing Rosetta

Rosetta implementaion could be testing using the Rosetta CLI.

Before we can start the service, we need to build the docker image:

```bash
make docker-build
```

Next, start the Testnet service by running:

```bash
make run-testnet
```

Wait until the node is done bootstrapping, then start the data check:

```bash
make check-testnet-data
```

Run the construction check:

```bash
make check-testnet-construction
```

## Rebuild the ContractInfoToken.go autogen file.

```bash
abigen --abi contractInfo.abi --pkg main --type ContractInfoToken --out client/contractInfoToken.go
```

## License

BSD 3-Clause

[1]: https://www.rosetta-api.org/
[2]: https://www.avalabs.org/
[3]: https://github.com/ava-labs/avalanchego
[4]: https://docs.avax.network/build/avalanchego-apis
[5]: https://docs.avax.network/build/tools/public-api
