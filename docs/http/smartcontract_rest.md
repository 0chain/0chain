# /0Chain/code/go/0chain.net/smartcontract/

### Module: 0chain.net/smartcontract


> Almost each package has the sc.go and the handler.go

```sh
File: 0Chain/code/go/0chain.net/smartcontract/faucetsc/sc.go
```

| Endpoint: fc.SmartContract.RestHandlers | Handler |
| ------ | ------ |
| /personalPeriodicLimit | fc.personalPeriodicLimit |
| /globalPerodicLimit | fc.globalPerodicLimit |
| /pourAmount | fc.pourAmount |
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
		if viper.GetBool(fmt.Sprintf("server_chain.smart_contract.%v", sc.GetName())) {
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
| /getPoolInfo | vsc.getPoolInfoHandler |
| /getClientPools | vsc.getClientPoolsHandler |

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



























