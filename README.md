# avalanche-rosetta

[Rosetta][1] server implementation for [Avalanche][2] Blockchain.

## Requirements

In order to run the Avalanche Rosetta server you will need access to [Gecko][3]
services via RPC. More info in available APIs found [here][4].

See Gecko documentation on how to run the chain node locally. If you don't run
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

| Method | Path                   | Status | Description
|--------|------------------------|--------|------------------------------------
| POST   | /network/list          | Y      | Get List of Available Networks
| POST   | /network/status        | Y      | Get Network Status
| POST   | /network/options       | Y      | Get Network Options
| POST   | /block                 | Y      | Get a Block
| POST   | /block/transaction     | Y      | Get a Block Transaction
| POST   | /account/balance       | Y      | Get an Account Balance
| POST   | /mempool               | Y      | Get All Mempool Transactions
| POST   | /mempool/transaction   | -      | Get a Mempool Transaction
| POST   | /construction/metadata | -      | Get Transaction Construction Metadata
| POST   | /construction/submit   | -      | Submit a Signed Transaction

### Development

Available Make commands:

- `make build` - Build the development version of the binary

## License

Apache License v2.0

[1]: https://www.rosetta-api.org/
[2]: https://www.avalabs.org/
[3]: https://github.com/ava-labs/gecko
[4]: https://docs.avax.network/v1.0/en/api/intro-apis/
[5]: https://figment.network/datahub/
