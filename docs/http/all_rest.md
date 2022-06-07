# /0Chain/code/go/0chain.net/smartcontract/

### Module: 0chain.net/smartcontract


> Almost each package has the sc.go and the handler.go

```sh
File: 0Chain/code/go/0chain.net/smartcontract/faucetsc/sc.go
```

| Endpoint: fc.SmartContract.RestHandlers | Handler |
| ------ | ------ |
| /personal-periodic-limit | fc.personalPeriodicLimit |
| /globalPerodicLimit | fc.globalPerodicLimit |
| /pour-amount | fc.pourAmount |
| /getConfig | fc.getConfigHandler |

| Endpoint: fc.SmartContractExecutionStats | Handler |
| ------ | ------ |
| updateLimits | metrics.GetOrRegisterTimer |
| pour | metrics.GetOrRegisterTimer |
| refill | metrics.GetOrRegisterTimer |
| tokens Poured | metrics.GetOrRegisterHistogram |
| token refills | metrics.GetOrRegisterHistogram |





```sh
File: 0Chain/code/go/0chain.net/smartcontract/interestpoolsc/sc.go
```

| Endpoint: fc.SmartContract.RestHandlers | Handler |
| ------ | ------ |
| /getPoolsStats | ipsc.getPoolsStats |
| /getLockConfig | ipsc.getLockConfig |


| Endpoint: fc.SmartContractExecutionStats | Handler |
| ------ | ------ |
| lock | metrics.GetOrRegisterTimer |
| unlock | metrics.GetOrRegisterTimer |
| updateVariables | metrics.GetOrRegisterTimer |




```sh
File: 0Chain/code/go/0chain.net/smartcontract/minersc/sc.go
```
> SetSC setting up smartcontract. implementing the interface

| Endpoint: fc.SmartContract.RestHandlers | Handler |
| ------ | ------ |
| /getNodepool | msc.GetNodepoolHandler |
| /getUserPools | msc.GetUserPoolsHandler |
| /getMinerList | msc.GetMinerListHandler |
| /getSharderList | msc.GetSharderListHandler |
| /getSharderKeepList | msc.GetSharderKeepListHandler |
| /getPhase | msc.GetPhaseHandler |
| /getDkgList | msc.GetDKGMinerListHandler |
| /getMpksList | msc.GetMinersMpksListHandler |
| /getGroupShareOrSigns | msc.GetGroupShareOrSignsHandler |
| /getMagicBlock | msc.GetMagicBlockHandler |
| /nodeStat | msc.nodeStatHandler |
| /nodePoolStat | msc.nodePoolStatHandler |
| /configs | msc.configsHandler |


| Endpoint: fc.SmartContractExecutionStats | Handler |
| ------ | ------ |
| add_miner | metrics.GetOrRegisterTimer |
| add_sharder | metrics.GetOrRegisterTimer |
| miner_health_check | metrics.GetOrRegisterTimer |
| sharder_health_check | metrics.GetOrRegisterTimer |
| update_settings | metrics.GetOrRegisterTimer |
| payFees | metrics.GetOrRegisterTimer |
| feesPaid | metrics.GetOrRegisterCounter |
| mintedTokens |metrics.GetOrRegisterCounter |


```sh
File: 0Chain/code/go/0chain.net/smartcontract/multisigsc/sc.go
```
> No REST Endpoints?


```sh
File: 0Chain/code/go/0chain.net/smartcontract/setupsc/setupsc.go
```

> It contains single function in a single file setupsc.go
```sh
func SetupSmartContracts() {
	for _, sc := range scs {
		if viper.GetBool(fmt.Sprintf("development.smart_contract.%v", sc.GetName())) {
			sc.InitSC()
			smartcontract.ContractMap[sc.GetAddress()] = sc
		}
	}
}
```

```sh
File: 0Chain/code/go/0chain.net/smartcontract/storagesc/sc.go
```
> sc configurations

| Endpoint: ssc.SmartContract.RestHandlers | Handler |
| ------ | ------ |
| /getConfig | ssc.getConfigHandler |

| Endpoint: fc.SmartContractExecutionStats | Handler |
| ------ | ------ |
| update_config | metrics.GetOrRegisterTimer |


> reading / writing

| Endpoint: ssc.SmartContract.RestHandlers | Handler |
| ------ | ------ |
| /latestreadmarker | ssc.LatestReadMarkerHandler |

| Endpoint: fc.SmartContractExecutionStats | Handler |
| ------ | ------ |
| read_redeem | metrics.GetOrRegisterTimer |
| commit_connection | metrics.GetOrRegisterTimer |


> allocation

| Endpoint: ssc.SmartContract.RestHandlers | Handler |
| ------ | ------ |
| /allocation | ssc.AllocationStatsHandler |
| /allocations | ssc.GetAllocationsHandler |
| /allocation_min_lock | ssc.GetAllocationMinLockHandler |

| Endpoint: fc.SmartContractExecutionStats | Handler |
| ------ | ------ |
| new_allocation_request | metrics.GetOrRegisterTimer |
| update_allocation_request | metrics.GetOrRegisterTimer |
| finalize_allocation | metrics.GetOrRegisterTimer |
| cancel_allocation | metrics.GetOrRegisterTimer |


> challenge

| Endpoint: ssc.SmartContract.RestHandlers | Handler |
| ------ | ------ |
| /openchallenges | ssc.OpenChallengeHandler |
| /getchallenge | ssc.GetChallengeHandler |

| Endpoint: fc.SmartContractExecutionStats | Handler |
| ------ | ------ |
| challenge_request | metrics.GetOrRegisterTimer |
| challenge_response | metrics.GetOrRegisterTimer |
| generate_challenges | metrics.GetOrRegisterTimer |


> validator

| Endpoint: fc.SmartContractExecutionStats | Handler |
| ------ | ------ |
| add_validator | metrics.GetOrRegisterTimer |


> validators stat (not function calls)

| Endpoint: fc.SmartContractExecutionStats | Handler |
| ------ | ------ |
| stat: add validator | metrics.GetOrRegisterCounter |
| stat: update validator | metrics.GetOrRegisterCounter |
| stat: number of validators | metrics.GetOrRegisterCounter |


> blobber

| Endpoint: ssc.SmartContract.RestHandlers | Handler |
| ------ | ------ |
| /getblobbers | ssc.GetBlobbersHandler |
| /getBlobber | ssc.GetBlobberHandler |

| Endpoint: fc.SmartContractExecutionStats | Handler |
| ------ | ------ |
| add_blobber |  metrics.GetOrRegisterTimer |
| update_blobber_settings |  metrics.GetOrRegisterTimer |


> blobber statistic (not function calls)


| Endpoint: fc.SmartContractExecutionStats | Handler |
| ------ | ------ |
| stat: number of blobbers | metrics.GetOrRegisterCounter |
| stat: add blobber | metrics.GetOrRegisterCounter |
| stat: update blobber | metrics.GetOrRegisterCounter |
| stat: remove blobber | metrics.GetOrRegisterCounter |


> read pool

| Endpoint: ssc.SmartContract.RestHandlers | Handler |
| ------ | ------ |
| /getReadPoolStat | ssc.getReadPoolStatHandler |
| /getReadPoolAllocBlobberStat | ssc.getReadPoolAllocBlobberStatHandler |

| Endpoint: fc.SmartContractExecutionStats | Handler |
| ------ | ------ |
| new_read_pool | metrics.GetOrRegisterTimer |
| read_pool_lock | metrics.GetOrRegisterTimer |
| read_pool_unlock | metrics.GetOrRegisterTimer |


> write pool

| Endpoint: ssc.SmartContract.RestHandlers | Handler |
| ------ | ------ |
| /getWritePoolStat | ssc.getWritePoolStatHandler |
| /getWritePoolAllocBlobberStat | ssc.getWritePoolAllocBlobberStatHandler |

| Endpoint: fc.SmartContractExecutionStats | Handler |
| ------ | ------ |
| write_pool_lock | metrics.GetOrRegisterTimer |
| write_pool_unlock | metrics.GetOrRegisterTimer |


> stake pool

| Endpoint: ssc.SmartContract.RestHandlers | Handler |
| ------ | ------ |
| /getStakePoolStat | ssc.getStakePoolStatHandler |
| /getUserStakePoolStat | ssc.getUserStakePoolStatHandler |

| Endpoint: fc.SmartContractExecutionStats | Handler |
| ------ | ------ |
| stake_pool_lock | metrics.GetOrRegisterTimer |
| stake_pool_unlock | metrics.GetOrRegisterTimer |
| stake_pool_pay_interests | metrics.GetOrRegisterTimer |


> challenge pool

| Endpoint: ssc.SmartContract.RestHandlers | Handler |
| ------ | ------ |
| /getChallengePoolStat | ssc.getChallengePoolStatHandler |


```sh
File: 0Chain/code/go/0chain.net/smartcontract/vestingsc/sc.go
```
> information (statistics) and configurations

| Endpoint: ssc.SmartContract.RestHandlers | Handler |
| ------ | ------ |
| /getConfig | vsc.getConfigHandler |
| /pool-info | vsc.getPoolInfoHandler |
| /client-pools | vsc.getClientPoolsHandler |

| Endpoint: fc.SmartContractExecutionStats | Handler |Comment|
| ------ | ------ | ------ |
| add | metrics.GetOrRegisterTimer |add {start,duration,lock_tokens,[destinations]}|
| delete | metrics.GetOrRegisterTimer | delete {start,duration,lock_tokens,[destinations]} |
| stop | metrics.GetOrRegisterTimer | stop vesting for a destination, unlocking all tokens released |
| unlock | metrics.GetOrRegisterTimer | tokens unlock for an existing pool (as owner, as a destination) |
| trigger | metrics.GetOrRegisterTimer | move vested tokens to destinations by pool owner |


```sh
File: 0Chain/code/go/0chain.net/smartcontract/zrc20sc/sc.go
```

| Endpoint: fc.SmartContractExecutionStats | Handler |
| ------ | ------ |
| createToken | metrics.GetOrRegisterTimer |
| digPool | metrics.GetOrRegisterTimer |
| fillPool | metrics.GetOrRegisterTimer |
| transferTo | metrics.GetOrRegisterTimer |
| drainPool | metrics.GetOrRegisterTimer |
| emptyPool | metrics.GetOrRegisterTimer |













# 0Chain/code/go/0chain.net/chaincore/

### Module: 0chain.net/chaincore

```sh
File: 0Chain/code/go/0chain.net/chaincore/chain/handler.go
```
> SetupHandlers sets up the necessary API end points

| Endpoint: http.HandleFunc | Handler |
| ------ | ------ |
| /v1/chain | GetChainHandler |
| /v1/chain/put | PutChainHandler |
| /v1/block | GetBlockHandler |
| /v1/block/latest-finalized | LatestFinalizedBlockHandler |
| /v1/block/latest-finalized-magic-block-summary | LatestFinalizedMagicBlockSummaryHandler |
| /v1/block/latest-finalized-magic-block | LatestFinalizedMagicBlockHandler |
| /v1/block/recent-finalized | RecentFinalizedBlockHandler |
| /v1/block/fee-stats | LatestBlockFeeStatsHandler |
| / | HomePageHandler |
| /_diagnostics | DiagnosticsHomepageHandler |
| /_diagnostics/dkg_process | DiagnosticsDKGHandler |
| /_diagnostics/round_info | RoundInfoHandler |
| /v1/transaction | PutTransaction |
| /_diagnostics/state_dump | StateDumpHandler |
| /v1/block/latest-finalized-ticket | LFBTicketHandler |

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
| /v1/_x2m/block/notarized-block | blockEntityMetadata |
| /v1/_x2m/block/state-change | blockStateChangeEntityMetadata |
| /v1/_x2m/state | partialStateEntityMetadata |
| /v1/_x2x/state/nodes | stateNodesEntityMetadata |

> SetupX2SRequestors

| Endpoint: node.RequestEntityHandler | Entity Metadata |
| ------ | ------ |
| /v1/block/latest-finalized-magic-block | blockEntityMetadata |


> SetupX2XResponders

| Endpoint: http.HandleFunc | Handler |
| ------ | ------ |
| /v1/_x2x/state/nodes | StateNodesHandler |


```sh
File: 0Chain/code/go/0chain.net/chaincore/chain/state_handler.go
```

> SetupStateHandlers - setup handlers to manage state

| Endpoint: http.HandleFunc | Handler |
| ------ | ------ |
| /v1/client/balance | c.GetBalanceHandler |
| /v1/scstate | c.GetNodeFromSCState |
| /v1/scstats/ | c.GetSCStats |
| /v1/screst/ | c.HandleSCRest |
| /_smart_contract_stats | c.SCStats |


```sh
File: 0Chain/code/go/0chain.net/chaincore/client/handler.go
```

> SetupHandlers sets up the necessary API end points

| Endpoint: http.HandleFunc | Handler |
| ------ | ------ |
| /v1/client | GetClientHandler |
| /v1/client | PutClient |


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
| /v1/_n2n/entity| datastore.PrintEntityHandler |

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












# 0Chain/code/go/0chain.net/miner

### Module: 0chain.net/miner

```sh
File: 0Chain/code/go/0chain.net/miner/handler.go
```
> SetupHandlers - setup miner handlers

| Endpoint: http.HandleFunc | Handler |
| ------ | ------ |
| /v1/chain/stats | ChainStatsHandler |
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
| /v1/_m2m/block/notarized-block |
| /v1/_m2m/block/verification_ticket |
| /v1/_m2m/block/notarization |


> SetupM2MReceivers - setup receivers for miner to miner communication

| Endpoint: http.HandleFunc | Handler: node.ToN2NReceiveEntityHandler |
| ------ | ------ |
| /v1/_m2m/round/vrf_share | VRFShareHandler |
| /v1/_m2m/block/verify | VerifyBlockHandler |
| /v1/_m2m/block/verification_ticket | VerificationTicketReceiptHandler |
| /v1/_m2m/block/notarization | NotarizationReceiptHandler |
| /v1/_m2m/block/notarized-block | NotarizedBlockHandler |


> SetupX2MResponders - setup responders

| Endpoint: http.HandleFunc | Handler: node.ToN2NSendEntityHandler |
| ------ | ------ |
| /v1/_x2m/block/notarized-block | NotarizedBlockSendHandler |
| /v1/_x2m/block/state-change | BlockStateChangeHandler |
| /v1/_x2m/state | PartialStateHandler |
| /v1/_m2m/dkg/share | SignShareRequestHandler |
| /v1/_m2m/chain/start | StartChainRequestHandler |

> SetupM2SRequestors - setup all requests to sharder by miner

| Endpoint: node.RequestEntityHandler | Entity Metadata |
| ------ | ------ |
| /v1/_m2s/block/latest-finalized | blockEntityMetadata |
| /v1/block | blockEntityMetadata |


> SetupM2MRequestors

| Endpoint: node.RequestEntityHandler | Entity Metadata |
| ------ | ------ |
| /v1/_m2m/dkg/share | dkgShareEntityMetadata |
| /v1/_m2m/chain/start | chainStartEntityMetadata |















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
| /v1/block | BlockHandler |
| /v1/block/magic/get | MagicBlockHandler |
| /v1/transaction/get/confirmation | TransactionConfirmationHandler |
| /v1/chain/stats | ChainStatsHandlerr |
| /_chain_stats | ChainStatsWriter |
| /_health_check | HealthCheckWriter |
| /v1/sharder/stats | SharderStatsHandler |

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
| /v1/_m2s/block/latest-finalized | LatestFinalizedBlockHandler |


```sh
File: 0Chain/code/go/0chain.net/sharder/s_handler.go
```


> SetupS2SRequestors

| Endpoint: node.RequestEntityHandler | Entity Metadata |
| ------ | ------ |
| /v1/_s2s/latest-round | roundEntityMetadata |
| /v1/_s2s/round/get | roundEntityMetadata |
| /v1/_s2s/block | blockEntityMetadata |
| /v1/_s2s/block-summary | blockSummaryEntityMetadata |
| /v1/_s2s/roundsummaries/get | roundSummariesEntityMetadata |
| /v1/_s2s/blocksummaries/get | blockSummariesEntityMetadata |


> SetupS2SResponders

| Endpoint: http.HandleFunc | Handler: node.ToN2NSendEntityHandler |
| ------ | ------ |
| /v1/_s2s/latest-round | LatestRoundRequestHandler |
| /v1/_s2s/round/get | RoundRequestHandler |
| /v1/_s2s/roundsummaries/get | RoundSummariesHandler) |
| /v1/_s2s/block | RoundBlockRequestHandler |
| /v1/_s2s/block-summary | BlockSummaryRequestHandler |
| /v1/_s2s/blocksummaries/get | BlockSummariesHandler |

> SetupX2SRespondes setups sharders responders for miner and sharders.
> BlockRequestHandler - used by nodes to get missing FB by received LFB
> ticket from sharder sent the ticket.

| Endpoint: http.HandleFunc | Handler: node.ToN2NSendEntityHandler |
| ------ | ------ |
| /v1/_s2s/block | RoundBlockRequestHandler |










# blobber/

### Module: blobbercore

```sh
File: blobber/code/go/0chain.net/blobbercore/handler/handler.go
```
> object operations

| Endpoint: http.HandleFunc | Handler |
| ------ | ------ |
| /v1/file/upload/{allocation} | UploadHandler |
| /v1/file/download/{allocation} | DownloadHandler |
| /v1/file/rename/{allocation} | RenameHandler |
| /v1/file/copy/{allocation} | CopyHandler |
| /v1/file/attributes/{allocation} | UpdateAttributesHandler |
| /v1/connection/commit/{allocation} | CommitHandler |
| /v1/file/commit-meta-txn/{allocation} | CommitMetaTxnHandler |
| /v1/file/collaborator/{allocation} | CollaboratorHandler |
| /v1/file/calculatehash/{allocation} | CalculateHashHandler |

> object info related apis

| Endpoint: http.HandleFunc | Handler |
| ------ | ------ |
| /allocation | AllocationHandlerr |
| /v1/file/download/{allocation} | FileMetaHandler |
| /v1/file/rename/{allocation} | FileStatsHandler |
| /v1/file/copy/{allocation} | ListHandler |
| /v1/file/attributes/{allocation} | ObjectPathHandler |
| /v1/connection/commit/{allocation} | ReferencePathHandler |
| /v1/file/commit-meta-txn/{allocation} | ObjectTreeHandler |


> admin related

| Endpoint: http.HandleFunc | Handler |
| ------ | ------ |
| /_debug | DumpGoRoutines |
| /_config | GetConfig |
| /_stats | stats.StatsHandler |
| /_statsJSON | stats.StatsJSONHandler |
| /_cleanupdisk | CleanupDiskHandler |
| /getstats | stats.GetStatsHandler |


### Module: validatorcore

```sh
File: blobber/code/go/0chain.net/validatorcore/storage/handler.go
```

> SetupHandlers sets up the necessary API end points

| Endpoint: http.HandleFunc | Handler |
| ------ | ------ |
| /v1/storage/challenge/new | ChallengeHandler |
| /debug | DumpGoRoutines |







