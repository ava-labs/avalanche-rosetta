# Postman Collection

During development, you can invoke the individual Rosetta endpoints using the Postman collection in `Avalanche-Rosetta.postman-collection.json`.

The folder also contains a little utility that can be used to sign transactions using a provided private key.

## Running transaction test signing utility

```sh
go run postman/test_signing_server.go --private-key PrivateKey-abcd123...
```

## Data and Construction Derive Endpoints

Both the data endpoints and construction derive endpoint requests are simple Postman requests. 

They are under Network, Block and Account folders in the collection, grouped by chain, and can be executed as-is. 

## Construction Endpoints
Construction Endpoints use Postman pre-request script and tests features to chain outputs of requests to one another. 

Requests can be found under `Construction` folder; grouped by chains first, then by transaction type.

The operations from the `/construction/process` request body are copied to the corresponding `/construction/payloads` request automatically.

Once the operations and preprocess metadata is set, the collection run feature of Postman can be used to execute the full transaction construction and broadcast, provided the test signing server is up and running as well.
