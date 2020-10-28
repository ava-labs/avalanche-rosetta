# avalanche-rosetta

[Rosetta][1] server implementation for [Avalanche][2] Blockchain.

*Under active development*

*Please note this implementation only works with C-Chain*

## Requirements

In order to run the Avalanche Rosetta server you will need access to [Avalanche][3]
services via RPC. More info in available APIs found [here][4].

See AvalancheGo documentation on how to run the chain node locally. If you don't run
the Avalanche node yourself you might use a hosted service like [Figment DataHub][5].

## Installation

*Not available yet*

## Usage

Before you start running the server you need to create a configuration file:

```json
{
  "rpc_endpoint": "https://testapi.avax.network",
  "listen_addr": "0.0.0.0:8080"
}

```

Start the server by running the following command:

```bash
avalanche-rosetta -config=./config.json
```

### RPC Endpoints

List of all available Rosetta RPC server endpoints

| Method | Path                     | Status | Description
|--------|--------------------------|--------|------------------------------------
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
| POST   | /construction/parse      | -      | Parse a Transaction
| POST   | /construction/payloads   | -      | Generate an Unsigned Transaction and Signing Payloads
| POST   | /construction/preprocess | -      | Create a Request to Fetch Metadata
| POST   | /construction/submit     | Y      | Submit a Signed Transaction

### Development

Available Make commands:

- `make build` - Build the development version of the binary

## License

Apache License v2.0

[1]: https://www.rosetta-api.org/
[2]: https://www.avalabs.org/
[3]: https://github.com/ava-labs/avalanchego
[4]: https://docs.avax.network/v1.0/en/api/intro-apis/
[5]: https://figment.io/datahub/
