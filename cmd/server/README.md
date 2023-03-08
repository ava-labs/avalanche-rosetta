# Avalanche Rosetta: some technical notes

Some notes to ease up maintenance handover

## Emerging architecture

Avalanche-Rosetta used to be centered around the C-chain. Support for P-Chain and X-chain (the latter to allow import/export tracking) has been added later on. To host these chains a structure is emerging:

- Clients: client package collects all the calls to AvalancheGo node backing Rosetta server. Note that to support P-chain, the indexer must be supported by the AvalancheGo node backing Rosetta. The indexer is required to poll P-chain block by height (C-chain supports that natively, P-chain does not).  
- Backends: backends of P-chain and X-chain pull information from client and implements the logic for the various services that Rosetta provides. C-chain has not (yet) a backend since its logic is still implemented at the service level. Creation of C-chain backend is left for a future refactoring.  
- Services: the entry points for the API that Rosetta provides. Services route the  request to the right backend (P-chain,  X-chain or fallback to C-chain logic not yet repackaged into its backend). Moreover it returns the backend response to client.  

## P-chain Block querying and Genesis special case

P-chain blocks are queried as follows:  

- by hash. In such case GetBlock API of platformVM is hit  
- by height. In such case the indexer is used. Note that the indexer returns the whole proposerVM blocks. So block retrieved by proposerVM is further processed to extract the P-chain inner block and only the latter is returned.

**P-chain Genesis Data must not be polled from APIs**.  The reasons are that:

- `NetworkStatus` endpoint must return Genesis ID and Timestamp if AvalancheGo P-chain client has not complete bootstrapping yet  
- If AvalancheGo P-chain node has not done bootstrapping, GetBlock endpoint cannot serve P-chain Genesis block.  

Genesis Block information is parsed directly from AvalancheGo codebase rather than be called from GetBlock endpoint.  

## Snowman++ handling

Snowman++ headers (aka proposerVM header) are not returned to client. P-chain blocks are referenced by:

- Their height  
- Their BlockID (not the full proposerVM block ID)  

ProposerVM headers are used only to retrieved timestamp for pre-Banff blocks. Note that indexer timestamp is never used as it cannot guarantee a deployment-independent information.  
