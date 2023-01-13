<div align="center">
  <img src="resources/camino-logo.png?raw=true">
</div>

---

# Camino Rosetta

[Rosetta][1] server implementation for [Camino][2] C-Chain.

## Requirements

In order to run the Camino Rosetta server you will need access to [Camino][3]
services via RPC. More info in available APIs found [here][4].

See CaminoGo documentation on how to run the chain node locally.

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
  "rpc_endpoint": "https://columbus.camino.network",
  "mode": "online",
  "listen_addr": "0.0.0.0:8080",
  "genesis_block_hash" :"0x62e22411c311431cac7d766334106134b708629d3ed24f22e259f67fa383712b",
  "network_name": "Columbus",
  "chain_id": 501
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
  "network_name": "Columbus",
  "chain_id": 501,
  "log_requests": true,
  "genesis_block_hash" :"0x62e22411c311431cac7d766334106134b708629d3ed24f22e259f67fa383712b",
  "index_unknown_tokens": false,
  "ingestion_mode" : "standard",
  "token_whitelist" : [],
  "validate_erc20_whitelist": false
}
```

Where:

| Name          | Type    | Default | Description
|---------------|---------|---------|-------------------------------------------
| mode          | string  | `online` | Mode of operations. One of: `online`, `offline`
| rpc_endpoint  | string  | `http://localhost:9650` | Camino RPC endpoint
| listen_addr   | string  | `http://localhost:8080` | Rosetta server listen address (host/port)
| network_name  | string  | -       | Camino network name
| chain_id      | integer | -       | Camino C-Chain ID
| genesis_block_hash    | string  | -         | The block hash for the genesis block
| index_unknown_tokens  | bool    | `false`   | Enables ingesting tokens that don't have a public symbol or decimal variable
| ingestion_mode        | string  | `standard`| Toggles between standard and analytics ingesting modes
| token_whitelist       |[]string | []        | Enables ingesting for the provided ERC20 contract addresses in standard mode.
| validate_erc20_whitelist  | bool | `false`  | Verifies provided ERC20 contract addresses in standard mode (node must be bootstrapped when rosetta server starts).

The token whitelist only supports tokens that emit evm transfer logs for all minting (from should be 0x000---), burning (to address should be 0x0000) and transfer events are supported.  All other tokens will break cause ingestion to fail.

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

Run the construction check for ERC-20s:

```bash
make check-testnet-construction-erc20
```

## Rebuild the ContractInfoToken.go autogen file.

```bash
abigen --abi contractInfo.abi --pkg main --type ContractInfoToken --out client/contractInfoToken.go
```

## License

BSD 3-Clause

[1]: https://www.rosetta-api.org/
[2]: https://camino.foundation/
[3]: https://github.com/chain4travel/caminogo
[4]: https://docs.camino.foundation/developer/apis/camino-node-apis/issuing-api-calls/
