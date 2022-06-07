# 0Chain/code/go/0chain.net/miner

### Module: 0chain.net/miner

```sh
File: 0Chain/code/go/0chain.net/miner/handler.go
```
> SetupHandlers - setup miner handlers

| Endpoint: http.HandleFunc | Handler |
| ------ | ------ |
| /v1/chain/get/stats | ChainStatsHandler |
| /_chain_stats | ChainStatsWriter |
| /_diagnostics/wallet_stats | GetWalletStats |
| /v1/miner/get/stats | MinerStatsHandler |


```sh
File: 0Chain/code/go/0chain.net/miner/miner/handler.go
```
> SetupHandlers - setup update config related handlers


| Endpoint: http.HandleFunc | Handler |
| ------ | ------ |
| /_hash | encryption.HashHandler |
| /_sign | encryption.SignHandler |
| updateConfigURL | ConfigUpdateHandler |
| updateConfigAllURL | ConfigUpdateAllHandler |


```sh
File: 0Chain/code/go/0chain.net/miner/m_handler.go
```

> SetupM2MSenders - setup senders for miner to miner communication

| Endpoint: node.SendEntityHandler |
| ------ |
| /v1/_m2m/round/vrf_share |
| /v1/_m2m/block/verify |
| /v1/_m2m/block/notarized_block |
| /v1/_m2m/block/verification_ticket |
| /v1/_m2m/block/notarization |


> SetupM2MReceivers - setup receivers for miner to miner communication

| Endpoint: http.HandleFunc | Handler: node.ToN2NReceiveEntityHandler |
| ------ | ------ |
| /v1/_m2m/round/vrf_share | VRFShareHandler |
| /v1/_m2m/block/verify | VerifyBlockHandler |
| /v1/_m2m/block/verification_ticket | VerificationTicketReceiptHandler |
| /v1/_m2m/block/notarization | NotarizationReceiptHandler |
| /v1/_m2m/block/notarized_block | NotarizedBlockHandler |


> SetupX2MResponders - setup responders

| Endpoint: http.HandleFunc | Handler: node.ToN2NSendEntityHandler |
| ------ | ------ |
| /v1/_x2m/block/notarized_block/get | NotarizedBlockSendHandler |
| /v1/_x2m/block/state_change/get | BlockStateChangeHandler |
| /v1/_x2m/state/get | PartialStateHandler |
| /v1/_m2m/dkg/share | SignShareRequestHandler |
| /v1/_m2m/chain/start | StartChainRequestHandler |

> SetupM2SRequestors - setup all requests to sharder by miner

| Endpoint: node.RequestEntityHandler | Entity Metadata |
| ------ | ------ |
| /v1/_m2s/block/latest_finalized/get | blockEntityMetadata |
| /v1/block | blockEntityMetadata |


> SetupM2MRequestors

| Endpoint: node.RequestEntityHandler | Entity Metadata |
| ------ | ------ |
| /v1/_m2m/dkg/share | dkgShareEntityMetadata |
| /v1/_m2m/chain/start | chainStartEntityMetadata |

