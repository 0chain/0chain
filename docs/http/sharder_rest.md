# 0Chain/code/go/0chain.net/sharder

### Module: 0chain.net/sharder

```sh
File: 0Chain/code/go/0chain.net/sharder/sharder/main.go
```
> SetupHandlers - setup miner handlers

| Endpoint: http.HandleFunc | Handler |
| ------ | ------ |
| /_hash | encryption.HashHandler |
| /_sign | encryption.SignHandler |


```sh
File: 0Chain/code/go/0chain.net/sharder/handler.go
```

> SetupHandlers sets up the necessary API end points

| Endpoint: http.HandleFunc | Handler |
| ------ | ------ |
| /v1/block/get | BlockHandler |
| /v1/block/magic/get | MagicBlockHandler |
| /v1/transaction/get/confirmation | TransactionConfirmationHandler |
| /v1/chain/get/stats | ChainStatsHandlerr |
| /_chain_stats | ChainStatsWriter |
| /_healthcheck | HealthCheckWriter |
| /v1/sharder/get/stats | SharderStatsHandler |

```sh
File: 0Chain/code/go/0chain.net/sharder/m_handler.go
```
> SetupM2SReceivers - setup handlers for all the messages received from the miner

| Endpoint: http.HandleFunc | Handler: node.ToN2NReceiveEntityHandler |
| ------ | ------ |
| /v1/_m2s/block/finalized | FinalizedBlockHandler |
| /v1/_m2s/block/notarized | NotarizedBlockHandler |
| /v1/_m2s/block/notarized/kick | NotarizedBlockKickHandler |

> SetupM2SResponders - setup handlers for all the requests from the miner

| Endpoint: http.HandleFunc | Handler: node.ToS2MSendEntityHandler |
| ------ | ------ |
| /v1/_m2s/block/latest_finalized/get | LatestFinalizedBlockHandler |


```sh
File: 0Chain/code/go/0chain.net/sharder/s_handler.go
```


> SetupS2SRequestors

| Endpoint: node.RequestEntityHandler | Entity Metadata |
| ------ | ------ |
| /v1/_s2s/latest_round/get | roundEntityMetadata |
| /v1/_s2s/round/get | roundEntityMetadata |
| /v1/_s2s/block/get | blockEntityMetadata |
| /v1/_s2s/blocksummary/get | blockSummaryEntityMetadata |
| /v1/_s2s/roundsummaries/get | roundSummariesEntityMetadata |
| /v1/_s2s/blocksummaries/get | blockSummariesEntityMetadata |


> SetupS2SResponders

| Endpoint: http.HandleFunc | Handler: node.ToN2NSendEntityHandler |
| ------ | ------ |
| /v1/_s2s/latest_round/get | LatestRoundRequestHandler |
| /v1/_s2s/round/get | RoundRequestHandler |
| /v1/_s2s/roundsummaries/get | RoundSummariesHandler) |
| /v1/_s2s/block/get | RoundBlockRequestHandler |
| /v1/_s2s/blocksummary/get | BlockSummaryRequestHandler |
| /v1/_s2s/blocksummaries/get | BlockSummariesHandler |

> SetupX2SRespondes setups sharders responders for miner and sharders.
> BlockRequestHandler - used by nodes to get missing FB by received LFB
> ticket from sharder sent the ticket.

| Endpoint: http.HandleFunc | Handler: node.ToN2NSendEntityHandler |
| ------ | ------ |
| /v1/_x2s/block/get | RoundBlockRequestHandler |






