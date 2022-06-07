# 0Chain/code/go/0chain.net/chaincore/

### Module: 0chain.net/chaincore

```sh
File: 0Chain/code/go/0chain.net/chaincore/chain/handler.go
```
> SetupHandlers sets up the necessary API end points

| Endpoint: http.HandleFunc | Handler |
| ------ | ------ |
| /v1/chain/get | GetChainHandler |
| /v1/chain/put | PutChainHandler |
| /v1/block | GetBlockHandler |
| /v1/block/latest-finalized | LatestFinalizedBlockHandler |
| /v1/block/latest_finalized_magic_block_summary | LatestFinalizedMagicBlockSummaryHandler |
| /v1/block/latest_finalized_magic_block | LatestFinalizedMagicBlockHandler |
| /v1/block/recent_finalized | RecentFinalizedBlockHandler |
| /v1/block/fee_stats | LatestBlockFeeStatsHandler |
| / | HomePageHandler |
| /_diagnostics | DiagnosticsHomepageHandler |
| /_diagnostics/dkg_process | DiagnosticsDKGHandler |
| /_diagnostics/round_info | RoundInfoHandler |
| /v1/transaction/put | PutTransaction |
| /_diagnostics/state_dump | StateDumpHandler |
| /v1/block/latest_finalized_ticket | LFBTicketHandler |

```sh
File: 0Chain/code/go/0chain.net/chaincore/chain/n2n_handler.go
```

> SetupNodeHandlers - setup the handlers for the chain

| Endpoint: http.HandleFunc | Handler |
| ------ | ------ |
| /_nh/list/m | c.GetMinersHandlerr |
| /_nh/list/s | c.GetShardersHandler |


> SetupX2SRequestors

| Endpoint: node.RequestEntityHandler | Entity Metadata |
| ------ | ------ |
| /v1/_x2m/block/notarized_block/get | blockEntityMetadata |
| /v1/_x2m/block/state_change/get | blockStateChangeEntityMetadata |
| /v1/_x2m/state/get | partialStateEntityMetadata |
| /v1/_x2x/state/get_nodes | stateNodesEntityMetadata |

> SetupX2SRequestors

| Endpoint: node.RequestEntityHandler | Entity Metadata |
| ------ | ------ |
| /v1/block/latest_finalized_magic_block | blockEntityMetadata |


> SetupX2XResponders

| Endpoint: http.HandleFunc | Handler |
| ------ | ------ |
| /v1/_x2x/state/get_nodes | StateNodesHandler |


```sh
File: 0Chain/code/go/0chain.net/chaincore/chain/state_handler.go
```

> SetupStateHandlers - setup handlers to manage state

| Endpoint: http.HandleFunc | Handler |
| ------ | ------ |
| /v1/client/get/balance | c.GetBalanceHandler |
| /v1/scstate/get | c.GetNodeFromSCState |
| /v1/scstats/ | c.GetSCStats |
| /v1/screst/ | c.HandleSCRest |
| /_smart_contract_stats | c.SCStats |


```sh
File: 0Chain/code/go/0chain.net/chaincore/client/handler.go
```

> SetupHandlers sets up the necessary API end points

| Endpoint: http.HandleFunc | Handler |
| ------ | ------ |
| /v1/client/get | GetClientHandler |
| /v1/client/put | PutClient |


```sh
File: 0Chain/code/go/0chain.net/chaincore/config/handler.go
```

> SetupHandlers - setup config related handlers

| Endpoint: http.HandleFunc | Handler |
| ------ | ------ |
| /v1/config/get | GetConfigHandler |


```sh
File: 0Chain/code/go/0chain.net/chaincore/diagnostics/handler.go
```

> SetupHandlers - setup diagnostics handlers

| Endpoint: http.HandleFunc | Handler |
| ------ | ------ |
| /_diagnostics/info | chain.InfoWriter |
| /v1/diagnostics/get/info | chain.InfoHandler |
| /_diagnostics/logs | logging.LogWriter |
| /_diagnostics/n2n_logs | logging.N2NLogWriter |
| /_diagnostics/mem_logs | logging.MemLogWriter |
| /_diagnostics/n2n/info | sc.N2NStatsWriter |
| /_diagnostics/miner_stats | sc.MinerStatsHandler |
| /_diagnostics/block_chain | sc.WIPBlockChainHandler |


```sh
File: 0Chain/code/go/0chain.net/chaincore/node/handler.go
```

> SetupHandlers - setup diagnostics handlers

| Endpoint: http.HandleFunc | Handler |
| ------ | ------ |
| /_nh/whoami | WhoAmIHandler |
| /_nh/status | StatusHandler |
| /_nh/getpoolmembers | GetPoolMembersHandler |


```sh
File: 0Chain/code/go/0chain.net/chaincore/node/n2n_handler.go
```

> etupN2NHandlers - Setup all the node 2 node communiations

| Endpoint: http.HandleFunc | Handler: ToN2NReceiveEntityHandler|
| ------ | ------ |
| /v1/_n2n/entity/post| datastore.PrintEntityHandler |

| Endpoint: http.HandleFunc | Handler: ToN2NSendEntityHandler |
| ------ | ------ |
| pullURL | PushToPullHandlerr |


```sh
File: 0Chain/code/go/0chain.net/chaincore/transaction/handler.go
```

> SetupHandlers - setup diagnostics handlers

| Endpoint: http.HandleFunc | Handler |
| ------ | ------ |
| /v1/transaction/get | GetTransaction |












