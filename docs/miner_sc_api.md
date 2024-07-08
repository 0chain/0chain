


# Miner Smart Contract Public API:
  

## Informations

### Version

0.1.0

## Content negotiation

### URI Schemes
  * https

### Consumes
  * application/json

### Produces
  * application/json

## All endpoints

###  miner_sc

| Method  | URI     | Name   | Summary |
|---------|---------|--------|---------|
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/delegate-rewards | [get delegate rewards](#get-delegate-rewards) | Get delegate rewards. |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/getDkgList | [get dkg list](#get-dkg-list) | Get DKG miners/sharder list. |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/getEvents | [get events](#get-events) | Get Events. |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/globalSettings | [get global settings](#get-global-settings) | Get global chain settings. |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/getGroupShareOrSigns | [get group share or signs](#get-group-share-or-signs) | Get group shares/signs. |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/hardfork | [get hardfork](#get-hardfork) | Get hardfork. |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/getMagicBlock | [get magic block](#get-magic-block) | Get magic block. |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/getMinerList | [get miner list](#get-miner-list) | Get Miner List. |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/configs | [get miner s c configs](#get-miner-s-c-configs) | Get Miner SC configs. |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/get_miners_stats | [get miners stats](#get-miners-stats) | Get miners stats. |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/getMpksList | [get mpks list](#get-mpks-list) | Get MPKs list. |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/nodeStat | [get node stat](#get-node-stat) | Get node stats. |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/getNodepool | [get nodepool](#get-nodepool) | Get Node Pool. |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/getPhase | [get phase](#get-phase) | Get phase node from the client state. |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/provider-rewards | [get provider rewards](#get-provider-rewards) | Get provider rewards. |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/getSharderKeepList | [get sharder keep list](#get-sharder-keep-list) | Get sharder keep list. |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/getSharderList | [get sharder list](#get-sharder-list) | Get Sharder List. |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/get_sharders_stats | [get sharders stats](#get-sharders-stats) | Get sharders stats. |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/getStakePoolStat | [get stake pool stat](#get-stake-pool-stat) | Get Stake Pool Stat. |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/getUserPools | [get user pools](#get-user-pools) | Get User Pools. |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/nodePoolStat | [node pool stat](#node-pool-stat) | Get node pool stats. |
| GET | /test/screst/nodeStat | [node stat operation](#node-stat-operation) |  |
  


## Paths

### <span id="get-delegate-rewards"></span> Get delegate rewards. (*GetDelegateRewards*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/delegate-rewards
```

Retrieve a list of delegate rewards satisfying the filter. Supports pagination.

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| end | `query` | string | `string` |  | ✓ |  | last block until which to get rewards |
| limit | `query` | string | `string` |  |  |  | limit for pagination |
| offset | `query` | string | `string` |  |  |  | offset for pagination |
| pool_id | `query` | string | `string` |  |  |  | ID of the delegate pool for which to get rewards |
| sort | `query` | string | `string` |  |  |  | Sort direction (desc or asc) |
| start | `query` | string | `string` |  | ✓ |  | start block from which to get rewards |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#get-delegate-rewards-200) | OK | RewardDelegate |  | [schema](#get-delegate-rewards-200-schema) |
| [400](#get-delegate-rewards-400) | Bad Request |  |  | [schema](#get-delegate-rewards-400-schema) |
| [500](#get-delegate-rewards-500) | Internal Server Error |  |  | [schema](#get-delegate-rewards-500-schema) |

#### Responses


##### <span id="get-delegate-rewards-200"></span> 200 - RewardDelegate
Status: OK

###### <span id="get-delegate-rewards-200-schema"></span> Schema
   
  

[][RewardDelegate](#reward-delegate)

##### <span id="get-delegate-rewards-400"></span> 400
Status: Bad Request

###### <span id="get-delegate-rewards-400-schema"></span> Schema

##### <span id="get-delegate-rewards-500"></span> 500
Status: Internal Server Error

###### <span id="get-delegate-rewards-500-schema"></span> Schema

### <span id="get-dkg-list"></span> Get DKG miners/sharder list. (*GetDkgList*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/getDkgList
```

Retrieve a list of the miners/sharders that are part of the DKG process, number of revealed shares and weither nodes are waiting.

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#get-dkg-list-200) | OK | DKGMinerNodes |  | [schema](#get-dkg-list-200-schema) |
| [500](#get-dkg-list-500) | Internal Server Error |  |  | [schema](#get-dkg-list-500-schema) |

#### Responses


##### <span id="get-dkg-list-200"></span> 200 - DKGMinerNodes
Status: OK

###### <span id="get-dkg-list-200-schema"></span> Schema
   
  

[DKGMinerNodes](#d-k-g-miner-nodes)

##### <span id="get-dkg-list-500"></span> 500
Status: Internal Server Error

###### <span id="get-dkg-list-500-schema"></span> Schema

### <span id="get-events"></span> Get Events. (*GetEvents*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/getEvents
```

Retrieve a list of events based on the filters, supports pagination.

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| block_number | `query` | string | `string` |  |  |  | block number where the event occurred |
| limit | `query` | string | `string` |  |  |  | limit for pagination |
| offset | `query` | string | `string` |  |  |  | offset for pagination |
| sort | `query` | string | `string` |  |  |  | Direction of sorting (desc or asc) |
| tag | `query` | string | `string` |  |  |  | tag of event |
| tx_hash | `query` | string | `string` |  |  |  | hash of transaction associated with the event |
| type | `query` | string | `string` |  |  |  | type of event |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#get-events-200) | OK | eventList |  | [schema](#get-events-200-schema) |
| [400](#get-events-400) | Bad Request |  |  | [schema](#get-events-400-schema) |

#### Responses


##### <span id="get-events-200"></span> 200 - eventList
Status: OK

###### <span id="get-events-200-schema"></span> Schema
   
  

[EventList](#event-list)

##### <span id="get-events-400"></span> 400
Status: Bad Request

###### <span id="get-events-400-schema"></span> Schema

### <span id="get-global-settings"></span> Get global chain settings. (*GetGlobalSettings*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/globalSettings
```

Retrieve global configuration object for the chain.

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#get-global-settings-200) | OK | MinerGlobalSettings |  | [schema](#get-global-settings-200-schema) |
| [400](#get-global-settings-400) | Bad Request |  |  | [schema](#get-global-settings-400-schema) |

#### Responses


##### <span id="get-global-settings-200"></span> 200 - MinerGlobalSettings
Status: OK

###### <span id="get-global-settings-200-schema"></span> Schema
   
  

[GlobalSettings](#global-settings)

##### <span id="get-global-settings-400"></span> 400
Status: Bad Request

###### <span id="get-global-settings-400-schema"></span> Schema

### <span id="get-group-share-or-signs"></span> Get group shares/signs. (*GetGroupShareOrSigns*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/getGroupShareOrSigns
```

Retrieve a list of group shares and signatures, part of DKG process. Read about it in View Change protocol in public docs.

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#get-group-share-or-signs-200) | OK | GroupSharesOrSigns |  | [schema](#get-group-share-or-signs-200-schema) |
| [400](#get-group-share-or-signs-400) | Bad Request |  |  | [schema](#get-group-share-or-signs-400-schema) |

#### Responses


##### <span id="get-group-share-or-signs-200"></span> 200 - GroupSharesOrSigns
Status: OK

###### <span id="get-group-share-or-signs-200-schema"></span> Schema
   
  

[GroupSharesOrSigns](#group-shares-or-signs)

##### <span id="get-group-share-or-signs-400"></span> 400
Status: Bad Request

###### <span id="get-group-share-or-signs-400-schema"></span> Schema

### <span id="get-hardfork"></span> Get hardfork. (*GetHardfork*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/hardfork
```

Retrieve hardfork information given its name, which is the round when it was applied.

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#get-hardfork-200) | OK | StringMap |  | [schema](#get-hardfork-200-schema) |
| [400](#get-hardfork-400) | Bad Request |  |  | [schema](#get-hardfork-400-schema) |
| [500](#get-hardfork-500) | Internal Server Error |  |  | [schema](#get-hardfork-500-schema) |

#### Responses


##### <span id="get-hardfork-200"></span> 200 - StringMap
Status: OK

###### <span id="get-hardfork-200-schema"></span> Schema
   
  

[StringMap](#string-map)

##### <span id="get-hardfork-400"></span> 400
Status: Bad Request

###### <span id="get-hardfork-400-schema"></span> Schema

##### <span id="get-hardfork-500"></span> 500
Status: Internal Server Error

###### <span id="get-hardfork-500-schema"></span> Schema

### <span id="get-magic-block"></span> Get magic block. (*GetMagicBlock*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/getMagicBlock
```

Retrieve the magic block, which is the first block in the beginning of each view change process, containing the information of the nodes contributing to the network (miners/sharders).

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#get-magic-block-200) | OK | MagicBlock |  | [schema](#get-magic-block-200-schema) |
| [400](#get-magic-block-400) | Bad Request |  |  | [schema](#get-magic-block-400-schema) |

#### Responses


##### <span id="get-magic-block-200"></span> 200 - MagicBlock
Status: OK

###### <span id="get-magic-block-200-schema"></span> Schema
   
  

[MagicBlock](#magic-block)

##### <span id="get-magic-block-400"></span> 400
Status: Bad Request

###### <span id="get-magic-block-400-schema"></span> Schema

### <span id="get-miner-list"></span> Get Miner List. (*GetMinerList*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/getMinerList
```

Retrieves a list of miners given the filters, supports pagination.

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| active | `query` | string | `string` |  |  |  | Whether the miner is active |
| killed | `query` | string | `string` |  |  |  | Whether the miner is killed |
| limit | `query` | string | `string` |  |  |  | limit for pagination |
| offset | `query` | string | `string` |  |  |  | offset for pagination |
| sort | `query` | string | `string` |  |  |  | direction of sorting (desc or asc) |
| stakable | `query` | string | `string` |  |  |  | Whether the miner is stakable |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#get-miner-list-200) | OK | InterfaceMap |  | [schema](#get-miner-list-200-schema) |
| [400](#get-miner-list-400) | Bad Request |  |  | [schema](#get-miner-list-400-schema) |
| [500](#get-miner-list-500) | Internal Server Error |  |  | [schema](#get-miner-list-500-schema) |

#### Responses


##### <span id="get-miner-list-200"></span> 200 - InterfaceMap
Status: OK

###### <span id="get-miner-list-200-schema"></span> Schema
   
  

[InterfaceMap](#interface-map)

##### <span id="get-miner-list-400"></span> 400
Status: Bad Request

###### <span id="get-miner-list-400-schema"></span> Schema

##### <span id="get-miner-list-500"></span> 500
Status: Internal Server Error

###### <span id="get-miner-list-500-schema"></span> Schema

### <span id="get-miner-s-c-configs"></span> Get Miner SC configs. (*GetMinerSCConfigs*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/configs
```

Retrieve the miner SC global configuration.

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#get-miner-s-c-configs-200) | OK | StringMap |  | [schema](#get-miner-s-c-configs-200-schema) |
| [400](#get-miner-s-c-configs-400) | Bad Request |  |  | [schema](#get-miner-s-c-configs-400-schema) |
| [500](#get-miner-s-c-configs-500) | Internal Server Error |  |  | [schema](#get-miner-s-c-configs-500-schema) |

#### Responses


##### <span id="get-miner-s-c-configs-200"></span> 200 - StringMap
Status: OK

###### <span id="get-miner-s-c-configs-200-schema"></span> Schema
   
  

[StringMap](#string-map)

##### <span id="get-miner-s-c-configs-400"></span> 400
Status: Bad Request

###### <span id="get-miner-s-c-configs-400-schema"></span> Schema

##### <span id="get-miner-s-c-configs-500"></span> 500
Status: Internal Server Error

###### <span id="get-miner-s-c-configs-500-schema"></span> Schema

### <span id="get-miners-stats"></span> Get miners stats. (*GetMinersStats*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/get_miners_stats
```

Retrieve statitics about the miners, including counts of active and inactive miners.

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#get-miners-stats-200) | OK | Int64Map |  | [schema](#get-miners-stats-200-schema) |
| [404](#get-miners-stats-404) | Not Found |  |  | [schema](#get-miners-stats-404-schema) |

#### Responses


##### <span id="get-miners-stats-200"></span> 200 - Int64Map
Status: OK

###### <span id="get-miners-stats-200-schema"></span> Schema
   
  

[Int64Map](#int64-map)

##### <span id="get-miners-stats-404"></span> 404
Status: Not Found

###### <span id="get-miners-stats-404-schema"></span> Schema

### <span id="get-mpks-list"></span> Get MPKs list. (*GetMpksList*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/getMpksList
```

Retrievs MPKs list of network nodes (miners/sharders).

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#get-mpks-list-200) | OK | Mpks |  | [schema](#get-mpks-list-200-schema) |
| [400](#get-mpks-list-400) | Bad Request |  |  | [schema](#get-mpks-list-400-schema) |

#### Responses


##### <span id="get-mpks-list-200"></span> 200 - Mpks
Status: OK

###### <span id="get-mpks-list-200-schema"></span> Schema
   
  

[Mpks](#mpks)

##### <span id="get-mpks-list-400"></span> 400
Status: Bad Request

###### <span id="get-mpks-list-400-schema"></span> Schema

### <span id="get-node-stat"></span> Get node stats. (*GetNodeStat*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/nodeStat
```

Retrieve the stats of a miner or sharder given the ID.

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| id | `query` | string | `string` |  | ✓ |  | miner or sharder ID |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#get-node-stat-200) | OK | nodeStat |  | [schema](#get-node-stat-200-schema) |
| [400](#get-node-stat-400) | Bad Request |  |  | [schema](#get-node-stat-400-schema) |
| [500](#get-node-stat-500) | Internal Server Error |  |  | [schema](#get-node-stat-500-schema) |

#### Responses


##### <span id="get-node-stat-200"></span> 200 - nodeStat
Status: OK

###### <span id="get-node-stat-200-schema"></span> Schema
   
  

[NodeStat](#node-stat)

##### <span id="get-node-stat-400"></span> 400
Status: Bad Request

###### <span id="get-node-stat-400-schema"></span> Schema

##### <span id="get-node-stat-500"></span> 500
Status: Internal Server Error

###### <span id="get-node-stat-500-schema"></span> Schema

### <span id="get-nodepool"></span> Get Node Pool. (*GetNodepool*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/getNodepool
```

Retrieve the node pool information for all the nodes in the network (miners/sharders).

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#get-nodepool-200) | OK | PoolMembersInfo |  | [schema](#get-nodepool-200-schema) |
| [400](#get-nodepool-400) | Bad Request |  |  | [schema](#get-nodepool-400-schema) |
| [500](#get-nodepool-500) | Internal Server Error |  |  | [schema](#get-nodepool-500-schema) |

#### Responses


##### <span id="get-nodepool-200"></span> 200 - PoolMembersInfo
Status: OK

###### <span id="get-nodepool-200-schema"></span> Schema
   
  

[PoolMembersInfo](#pool-members-info)

##### <span id="get-nodepool-400"></span> 400
Status: Bad Request

###### <span id="get-nodepool-400-schema"></span> Schema

##### <span id="get-nodepool-500"></span> 500
Status: Internal Server Error

###### <span id="get-nodepool-500-schema"></span> Schema

### <span id="get-phase"></span> Get phase node from the client state. (*GetPhase*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/getPhase
```

Phase node has information about the current phase of the network, including the current round, and number of restarts.

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#get-phase-200) | OK | PhaseNode |  | [schema](#get-phase-200-schema) |
| [400](#get-phase-400) | Bad Request |  |  | [schema](#get-phase-400-schema) |

#### Responses


##### <span id="get-phase-200"></span> 200 - PhaseNode
Status: OK

###### <span id="get-phase-200-schema"></span> Schema
   
  

[PhaseNode](#phase-node)

##### <span id="get-phase-400"></span> 400
Status: Bad Request

###### <span id="get-phase-400-schema"></span> Schema

### <span id="get-provider-rewards"></span> Get provider rewards. (*GetProviderRewards*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/provider-rewards
```

Retrieve list of provider rewards satisfying filter, supports pagination.

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| end | `query` | string | `string` |  | ✓ |  | end time of interval |
| id | `query` | string | `string` |  |  |  | ID of the provider for which to get rewards |
| limit | `query` | string | `string` |  |  |  | limit for pagination |
| offset | `query` | string | `string` |  |  |  | offset for pagination |
| sort | `query` | string | `string` |  |  |  | Sort direction (desc or asc) |
| start | `query` | string | `string` |  | ✓ |  | start time of interval |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#get-provider-rewards-200) | OK | RewardProvider |  | [schema](#get-provider-rewards-200-schema) |
| [400](#get-provider-rewards-400) | Bad Request |  |  | [schema](#get-provider-rewards-400-schema) |
| [500](#get-provider-rewards-500) | Internal Server Error |  |  | [schema](#get-provider-rewards-500-schema) |

#### Responses


##### <span id="get-provider-rewards-200"></span> 200 - RewardProvider
Status: OK

###### <span id="get-provider-rewards-200-schema"></span> Schema
   
  

[][RewardProvider](#reward-provider)

##### <span id="get-provider-rewards-400"></span> 400
Status: Bad Request

###### <span id="get-provider-rewards-400-schema"></span> Schema

##### <span id="get-provider-rewards-500"></span> 500
Status: Internal Server Error

###### <span id="get-provider-rewards-500-schema"></span> Schema

### <span id="get-sharder-keep-list"></span> Get sharder keep list. (*GetSharderKeepList*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/getSharderKeepList
```

Retrieve a list of sharders in the keep list.

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#get-sharder-keep-list-200) | OK | MinerNodes |  | [schema](#get-sharder-keep-list-200-schema) |
| [500](#get-sharder-keep-list-500) | Internal Server Error |  |  | [schema](#get-sharder-keep-list-500-schema) |

#### Responses


##### <span id="get-sharder-keep-list-200"></span> 200 - MinerNodes
Status: OK

###### <span id="get-sharder-keep-list-200-schema"></span> Schema
   
  

[MinerNodes](#miner-nodes)

##### <span id="get-sharder-keep-list-500"></span> 500
Status: Internal Server Error

###### <span id="get-sharder-keep-list-500-schema"></span> Schema

### <span id="get-sharder-list"></span> Get Sharder List. (*GetSharderList*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/getSharderList
```

Retrieves a list of sharders based on the filters, supports pagination.

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| active | `query` | string | `string` |  |  |  | Whether the sharder is active |
| killed | `query` | string | `string` |  |  |  | Whether the sharder is killed |
| limit | `query` | string | `string` |  |  |  | limit for pagination |
| offset | `query` | string | `string` |  |  |  | offset for pagination |
| sort | `query` | string | `string` |  |  |  | Direction of sorting (desc or asc) |
| stakable | `query` | string | `string` |  |  |  | Whether the sharder is stakable |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#get-sharder-list-200) | OK | InterfaceMap |  | [schema](#get-sharder-list-200-schema) |
| [400](#get-sharder-list-400) | Bad Request |  |  | [schema](#get-sharder-list-400-schema) |
| [500](#get-sharder-list-500) | Internal Server Error |  |  | [schema](#get-sharder-list-500-schema) |

#### Responses


##### <span id="get-sharder-list-200"></span> 200 - InterfaceMap
Status: OK

###### <span id="get-sharder-list-200-schema"></span> Schema
   
  

[InterfaceMap](#interface-map)

##### <span id="get-sharder-list-400"></span> 400
Status: Bad Request

###### <span id="get-sharder-list-400-schema"></span> Schema

##### <span id="get-sharder-list-500"></span> 500
Status: Internal Server Error

###### <span id="get-sharder-list-500-schema"></span> Schema

### <span id="get-sharders-stats"></span> Get sharders stats. (*GetShardersStats*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/get_sharders_stats
```

Retreive statistics about the sharders, including counts of active and inactive sharders.

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#get-sharders-stats-200) | OK | Int64Map |  | [schema](#get-sharders-stats-200-schema) |
| [404](#get-sharders-stats-404) | Not Found |  |  | [schema](#get-sharders-stats-404-schema) |

#### Responses


##### <span id="get-sharders-stats-200"></span> 200 - Int64Map
Status: OK

###### <span id="get-sharders-stats-200-schema"></span> Schema
   
  

[Int64Map](#int64-map)

##### <span id="get-sharders-stats-404"></span> 404
Status: Not Found

###### <span id="get-sharders-stats-404-schema"></span> Schema

### <span id="get-stake-pool-stat"></span> Get Stake Pool Stat. (*GetStakePoolStat*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/getStakePoolStat
```

Retrieve statistic for all locked tokens of a stake pool.

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| provider_id | `query` | string | `string` |  | ✓ |  | id of a provider |
| provider_type | `query` | string | `string` |  | ✓ |  | type of the provider, possible values are: miner. sharder |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#get-stake-pool-stat-200) | OK | stakePoolStat |  | [schema](#get-stake-pool-stat-200-schema) |
| [400](#get-stake-pool-stat-400) | Bad Request |  |  | [schema](#get-stake-pool-stat-400-schema) |
| [500](#get-stake-pool-stat-500) | Internal Server Error |  |  | [schema](#get-stake-pool-stat-500-schema) |

#### Responses


##### <span id="get-stake-pool-stat-200"></span> 200 - stakePoolStat
Status: OK

###### <span id="get-stake-pool-stat-200-schema"></span> Schema
   
  

[StakePoolStat](#stake-pool-stat)

##### <span id="get-stake-pool-stat-400"></span> 400
Status: Bad Request

###### <span id="get-stake-pool-stat-400-schema"></span> Schema

##### <span id="get-stake-pool-stat-500"></span> 500
Status: Internal Server Error

###### <span id="get-stake-pool-stat-500-schema"></span> Schema

### <span id="get-user-pools"></span> Get User Pools. (*GetUserPools*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/getUserPools
```

Retrieve user stake pools, supports pagination.

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| client_id | `query` | string | `string` |  | ✓ |  | client for which to get user stake pools |
| limit | `query` | string | `string` |  |  |  | pagination limit |
| offset | `query` | string | `string` |  |  |  | pagination offset |
| sort | `query` | string | `string` |  |  |  | sorting direction (desc or asc) based on pool id and type. |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#get-user-pools-200) | OK | userPoolStat |  | [schema](#get-user-pools-200-schema) |
| [400](#get-user-pools-400) | Bad Request |  |  | [schema](#get-user-pools-400-schema) |
| [500](#get-user-pools-500) | Internal Server Error |  |  | [schema](#get-user-pools-500-schema) |

#### Responses


##### <span id="get-user-pools-200"></span> 200 - userPoolStat
Status: OK

###### <span id="get-user-pools-200-schema"></span> Schema
   
  

[UserPoolStat](#user-pool-stat)

##### <span id="get-user-pools-400"></span> 400
Status: Bad Request

###### <span id="get-user-pools-400-schema"></span> Schema

##### <span id="get-user-pools-500"></span> 500
Status: Internal Server Error

###### <span id="get-user-pools-500-schema"></span> Schema

### <span id="node-pool-stat"></span> Get node pool stats. (*NodePoolStat*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/nodePoolStat
```

Retrieves node stake pool stats for a given client, given the id of the client and the node.

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| id | `query` | string | `string` |  | ✓ |  | miner/sharder node ID |
| pool_id | `query` | string | `string` |  |  |  | pool_id |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#node-pool-stat-200) | OK | NodePool |  | [schema](#node-pool-stat-200-schema) |
| [400](#node-pool-stat-400) | Bad Request |  |  | [schema](#node-pool-stat-400-schema) |
| [500](#node-pool-stat-500) | Internal Server Error |  |  | [schema](#node-pool-stat-500-schema) |

#### Responses


##### <span id="node-pool-stat-200"></span> 200 - NodePool
Status: OK

###### <span id="node-pool-stat-200-schema"></span> Schema
   
  

[][NodePool](#node-pool)

##### <span id="node-pool-stat-400"></span> 400
Status: Bad Request

###### <span id="node-pool-stat-400-schema"></span> Schema

##### <span id="node-pool-stat-500"></span> 500
Status: Internal Server Error

###### <span id="node-pool-stat-500-schema"></span> Schema

### <span id="node-stat-operation"></span> node stat operation (*nodeStatOperation*)

```
GET /test/screst/nodeStat
```

lists sharders

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| id | `query` | string | `string` |  | ✓ |  | miner or sharder ID |
| include_delegates | `query` | string | `string` |  |  |  | set to "true" if the delegate pools are required as well |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#node-stat-operation-200) | OK | nodeStat |  | [schema](#node-stat-operation-200-schema) |
| [400](#node-stat-operation-400) | Bad Request |  |  | [schema](#node-stat-operation-400-schema) |
| [500](#node-stat-operation-500) | Internal Server Error |  |  | [schema](#node-stat-operation-500-schema) |

#### Responses


##### <span id="node-stat-operation-200"></span> 200 - nodeStat
Status: OK

###### <span id="node-stat-operation-200-schema"></span> Schema
   
  

[NodeStat](#node-stat)

##### <span id="node-stat-operation-400"></span> 400
Status: Bad Request

###### <span id="node-stat-operation-400-schema"></span> Schema

##### <span id="node-stat-operation-500"></span> 500
Status: Internal Server Error

###### <span id="node-stat-operation-500-schema"></span> Schema

## Models

### <span id="allocation"></span> Allocation


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| AllocationID | string| `string` |  | |  |  |
| Cancelled | boolean| `bool` |  | |  |  |
| CreatedAt | date-time (formatted string)| `strfmt.DateTime` |  | |  |  |
| DataShards | int64 (formatted integer)| `int64` |  | |  |  |
| Expiration | int64 (formatted integer)| `int64` |  | |  |  |
| FailedChallenges | int64 (formatted integer)| `int64` |  | |  |  |
| FileOptions | uint16 (formatted integer)| `uint16` |  | |  |  |
| Finalized | boolean| `bool` |  | |  |  |
| ID | uint64 (formatted integer)| `uint64` |  | |  |  |
| LatestClosedChallengeTxn | string| `string` |  | |  |  |
| NumReads | int64 (formatted integer)| `int64` |  | |  |  |
| NumWrites | int64 (formatted integer)| `int64` |  | |  |  |
| OpenChallenges | int64 (formatted integer)| `int64` |  | |  |  |
| Owner | string| `string` |  | |  |  |
| OwnerPublicKey | string| `string` |  | |  |  |
| ParityShards | int64 (formatted integer)| `int64` |  | |  |  |
| Size | int64 (formatted integer)| `int64` |  | |  |  |
| StartTime | int64 (formatted integer)| `int64` |  | |  |  |
| SuccessfulChallenges | int64 (formatted integer)| `int64` |  | |  |  |
| Terms | [][AllocationBlobberTerm](#allocation-blobber-term)| `[]*AllocationBlobberTerm` |  | |  |  |
| ThirdPartyExtendable | boolean| `bool` |  | |  |  |
| TimeUnit | int64 (formatted integer)| `int64` |  | |  |  |
| TotalChallenges | int64 (formatted integer)| `int64` |  | |  |  |
| TransactionID | string| `string` |  | |  |  |
| UpdatedAt | date-time (formatted string)| `strfmt.DateTime` |  | |  |  |
| UsedSize | int64 (formatted integer)| `int64` |  | |  |  |
| User | [User](#user)| `User` |  | |  |  |
| moved_back | [Coin](#coin)| `Coin` |  | |  |  |
| moved_to_challenge | [Coin](#coin)| `Coin` |  | |  |  |
| moved_to_validators | [Coin](#coin)| `Coin` |  | |  |  |
| read_price_max | [Coin](#coin)| `Coin` |  | |  |  |
| read_price_min | [Coin](#coin)| `Coin` |  | |  |  |
| write_pool | [Coin](#coin)| `Coin` |  | |  |  |
| write_price_max | [Coin](#coin)| `Coin` |  | |  |  |
| write_price_min | [Coin](#coin)| `Coin` |  | |  |  |



### <span id="allocation-blobber-term"></span> AllocationBlobberTerm


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| AllocBlobberIdx | int64 (formatted integer)| `int64` |  | |  |  |
| AllocationID | int64 (formatted integer)| `int64` |  | |  |  |
| AllocationIdHash | string| `string` |  | |  |  |
| BlobberID | string| `string` |  | |  |  |
| CreatedAt | date-time (formatted string)| `strfmt.DateTime` |  | |  |  |
| DeletedAt | [DeletedAt](#deleted-at)| `DeletedAt` |  | |  |  |
| ID | uint64 (formatted integer)| `uint64` |  | |  |  |
| ReadPrice | int64 (formatted integer)| `int64` |  | |  |  |
| UpdatedAt | date-time (formatted string)| `strfmt.DateTime` |  | |  |  |
| WritePrice | int64 (formatted integer)| `int64` |  | |  |  |



### <span id="approved-minter"></span> ApprovedMinter


  

| Name | Type | Go type | Default | Description | Example |
|------|------|---------| ------- |-------------|---------|
| ApprovedMinter | int64 (formatted integer)| int64 | |  |  |



### <span id="authorizer-aggregate"></span> AuthorizerAggregate


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| AuthorizerID | string| `string` |  | |  |  |
| CreatedAt | date-time (formatted string)| `strfmt.DateTime` |  | |  |  |
| ID | uint64 (formatted integer)| `uint64` |  | |  |  |
| IsKilled | boolean| `bool` |  | |  |  |
| IsShutdown | boolean| `bool` |  | |  |  |
| Round | int64 (formatted integer)| `int64` |  | |  |  |
| ServiceCharge | double (formatted number)| `float64` |  | |  |  |
| URL | string| `string` |  | |  |  |
| fee | [Coin](#coin)| `Coin` |  | |  |  |
| last_health_check | [Timestamp](#timestamp)| `Timestamp` |  | |  |  |
| total_burn | [Coin](#coin)| `Coin` |  | |  |  |
| total_mint | [Coin](#coin)| `Coin` |  | |  |  |
| total_rewards | [Coin](#coin)| `Coin` |  | |  |  |
| total_stake | [Coin](#coin)| `Coin` |  | |  |  |



### <span id="authorizer-snapshot"></span> AuthorizerSnapshot


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| AuthorizerID | string| `string` |  | |  |  |
| CreationRound | int64 (formatted integer)| `int64` |  | |  |  |
| IsKilled | boolean| `bool` |  | |  |  |
| IsShutdown | boolean| `bool` |  | |  |  |
| Round | int64 (formatted integer)| `int64` |  | |  |  |
| ServiceCharge | double (formatted number)| `float64` |  | |  |  |
| fee | [Coin](#coin)| `Coin` |  | |  |  |
| total_burn | [Coin](#coin)| `Coin` |  | |  |  |
| total_mint | [Coin](#coin)| `Coin` |  | |  |  |
| total_rewards | [Coin](#coin)| `Coin` |  | |  |  |
| total_stake | [Coin](#coin)| `Coin` |  | |  |  |



### <span id="blobber-aggregate"></span> BlobberAggregate


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| Allocated | int64 (formatted integer)| `int64` |  | |  |  |
| BlobberID | string| `string` |  | |  |  |
| Capacity | int64 (formatted integer)| `int64` |  | |  |  |
| ChallengesCompleted | uint64 (formatted integer)| `uint64` |  | |  |  |
| ChallengesPassed | uint64 (formatted integer)| `uint64` |  | |  |  |
| CreatedAt | date-time (formatted string)| `strfmt.DateTime` |  | |  |  |
| Downtime | uint64 (formatted integer)| `uint64` |  | |  |  |
| ID | uint64 (formatted integer)| `uint64` |  | |  |  |
| InactiveRounds | int64 (formatted integer)| `int64` |  | |  |  |
| IsKilled | boolean| `bool` |  | |  |  |
| IsRestricted | boolean| `bool` |  | |  |  |
| IsShutdown | boolean| `bool` |  | |  |  |
| NotAvailable | boolean| `bool` |  | |  |  |
| OpenChallenges | uint64 (formatted integer)| `uint64` |  | |  |  |
| RankMetric | double (formatted number)| `float64` |  | |  |  |
| ReadData | int64 (formatted integer)| `int64` |  | |  |  |
| Round | int64 (formatted integer)| `int64` |  | |  |  |
| SavedData | int64 (formatted integer)| `int64` |  | |  |  |
| ServiceCharge | double (formatted number)| `float64` |  | |  |  |
| URL | string| `string` |  | |  |  |
| last_health_check | [Timestamp](#timestamp)| `Timestamp` |  | |  |  |
| offers_total | [Coin](#coin)| `Coin` |  | |  |  |
| total_block_rewards | [Coin](#coin)| `Coin` |  | |  |  |
| total_read_income | [Coin](#coin)| `Coin` |  | |  |  |
| total_rewards | [Coin](#coin)| `Coin` |  | |  |  |
| total_slashed_stake | [Coin](#coin)| `Coin` |  | |  |  |
| total_stake | [Coin](#coin)| `Coin` |  | |  |  |
| total_storage_income | [Coin](#coin)| `Coin` |  | |  |  |
| write_price | [Coin](#coin)| `Coin` |  | |  |  |



### <span id="blobber-snapshot"></span> BlobberSnapshot


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| Allocated | int64 (formatted integer)| `int64` |  | |  |  |
| BlobberID | string| `string` |  | |  |  |
| Capacity | int64 (formatted integer)| `int64` |  | |  |  |
| ChallengesCompleted | uint64 (formatted integer)| `uint64` |  | |  |  |
| ChallengesPassed | uint64 (formatted integer)| `uint64` |  | |  |  |
| CreationRound | int64 (formatted integer)| `int64` |  | |  |  |
| IsKilled | boolean| `bool` |  | |  |  |
| IsShutdown | boolean| `bool` |  | |  |  |
| OpenChallenges | uint64 (formatted integer)| `uint64` |  | |  |  |
| RankMetric | double (formatted number)| `float64` |  | |  |  |
| ReadData | int64 (formatted integer)| `int64` |  | |  |  |
| Round | int64 (formatted integer)| `int64` |  | |  |  |
| SavedData | int64 (formatted integer)| `int64` |  | |  |  |
| offers_total | [Coin](#coin)| `Coin` |  | |  |  |
| total_block_rewards | [Coin](#coin)| `Coin` |  | |  |  |
| total_read_income | [Coin](#coin)| `Coin` |  | |  |  |
| total_rewards | [Coin](#coin)| `Coin` |  | |  |  |
| total_service_charge | [Coin](#coin)| `Coin` |  | |  |  |
| total_slashed_stake | [Coin](#coin)| `Coin` |  | |  |  |
| total_stake | [Coin](#coin)| `Coin` |  | |  |  |
| total_storage_income | [Coin](#coin)| `Coin` |  | |  |  |
| write_price | [Coin](#coin)| `Coin` |  | |  |  |



### <span id="block"></span> Block


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| ChainId | string| `string` |  | |  |  |
| CreatedAt | date-time (formatted string)| `strfmt.DateTime` |  | |  |  |
| CreationDate | int64 (formatted integer)| `int64` |  | |  |  |
| Hash | string| `string` |  | |  |  |
| ID | uint64 (formatted integer)| `uint64` |  | |  |  |
| MagicBlockHash | string| `string` |  | |  |  |
| MerkleTreeRoot | string| `string` |  | |  |  |
| MinerID | string| `string` |  | |  |  |
| NumTxns | int64 (formatted integer)| `int64` |  | |  |  |
| PrevHash | string| `string` |  | |  |  |
| ReceiptMerkleTreeRoot | string| `string` |  | |  |  |
| Round | int64 (formatted integer)| `int64` |  | |  |  |
| RoundRandomSeed | int64 (formatted integer)| `int64` |  | |  |  |
| RoundTimeoutCount | int64 (formatted integer)| `int64` |  | |  |  |
| RunningTxnCount | string| `string` |  | |  |  |
| Signature | string| `string` |  | |  |  |
| StateChangesCount | int64 (formatted integer)| `int64` |  | |  |  |
| StateHash | string| `string` |  | |  |  |
| UpdatedAt | date-time (formatted string)| `strfmt.DateTime` |  | |  |  |
| Version | string| `string` |  | |  |  |



### <span id="block-summary"></span> BlockSummary


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| Hash | string| `string` |  | |  |  |
| K | int64 (formatted integer)| `int64` |  | |  |  |
| MagicBlockNumber | int64 (formatted integer)| `int64` |  | |  |  |
| MerkleTreeRoot | string| `string` |  | |  |  |
| MinerID | string| `string` |  | |  |  |
| N | int64 (formatted integer)| `int64` |  | |  |  |
| NumTxns | int64 (formatted integer)| `int64` |  | |  |  |
| PreviousMagicBlockHash | string| `string` |  | |  |  |
| ReceiptMerkleTreeRoot | string| `string` |  | |  |  |
| Round | int64 (formatted integer)| `int64` |  | |  |  |
| RoundRandomSeed | int64 (formatted integer)| `int64` |  | |  |  |
| StartingRound | int64 (formatted integer)| `int64` |  | |  |  |
| StateChangesCount | int64 (formatted integer)| `int64` |  | |  |  |
| T | int64 (formatted integer)| `int64` |  | |  |  |
| Version | string| `string` |  | | Version of the entity |  |
| creation_date | [Timestamp](#timestamp)| `Timestamp` |  | |  |  |
| miners | [Pool](#pool)| `Pool` |  | |  |  |
| mpks | [Mpks](#mpks)| `Mpks` |  | |  |  |
| sharders | [Pool](#pool)| `Pool` |  | |  |  |
| share_or_signs | [GroupSharesOrSigns](#group-shares-or-signs)| `GroupSharesOrSigns` |  | |  |  |
| state_hash | [Key](#key)| `Key` |  | |  |  |



### <span id="chain-stats"></span> ChainStats


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| Count | int64 (formatted integer)| `int64` |  | | Number of finalized blocks generated in the block chain since genesis. |  |
| CurrentRound | int64 (formatted integer)| `int64` |  | | The number that represents the current round of the blockchain. |  |
| LastFinalizedRound | int64 (formatted integer)| `int64` |  | | The number that represents the round that generated the latest finalized block. |  |
| Max | double (formatted number)| `float64` |  | | Maximum finalization time of a block, in milliseconds. |  |
| Mean | double (formatted number)| `float64` |  | | Mean (Average) finalization time of a block, in milliseconds. |  |
| Min | double (formatted number)| `float64` |  | | Minimum finalization time of a block, in milliseconds. |  |
| Percentile50 | double (formatted number)| `float64` |  | | The block finalization time value, in milliseconds, which the specified percentage of block finalization events lie below. |  |
| Percentile90 | double (formatted number)| `float64` |  | | The block finalization time value, in milliseconds, which the specified percentage of block finalization events lie below. |  |
| Percentile95 | double (formatted number)| `float64` |  | | The block finalization time value, in milliseconds, which the specified percentage of block finalization events lie below. |  |
| Percentile99 | double (formatted number)| `float64` |  | | The block finalization time value, in milliseconds, which the specified percentage of block finalization events lie below. |  |
| Rate1 | double (formatted number)| `float64` |  | | The moving average rate of occurrence of block finalization events per second during the specified time window. |  |
| Rate15 | double (formatted number)| `float64` |  | | The moving average rate of occurrence of block finalization events per second during the specified time window. |  |
| Rate5 | double (formatted number)| `float64` |  | | The moving average rate of occurrence of block finalization events per second during the specified time window. |  |
| RateMean | double (formatted number)| `float64` |  | | The overall mean rate of occurrence of block finalization events per second. |  |
| RunningTxnCount | int64 (formatted integer)| `int64` |  | | The total count of all transactions included in all the blocks generated by the blockchain. |  |
| StdDev | double (formatted number)| `float64` |  | | Standard deviation of the finalization time of a block from the mean number, in milliseconds. |  |
| delta | [Duration](#duration)| `Duration` |  | |  |  |



### <span id="challenge"></span> Challenge


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| AllocationID | string| `string` |  | |  |  |
| AllocationRoot | string| `string` |  | |  |  |
| BlobberID | string| `string` |  | |  |  |
| ChallengeID | string| `string` |  | |  |  |
| CreatedAt | [Timestamp](#timestamp)| `Timestamp` |  | |  |  |
| ExpiredN | int64 (formatted integer)| `int64` |  | |  |  |
| ID | uint64 (formatted integer)| `uint64` |  | |  |  |
| Passed | boolean| `bool` |  | |  |  |
| Responded | int64 (formatted integer)| `int64` |  | |  |  |
| RoundCreatedAt | int64 (formatted integer)| `int64` |  | |  |  |
| RoundResponded | int64 (formatted integer)| `int64` |  | |  |  |
| Seed | int64 (formatted integer)| `int64` |  | |  |  |
| UpdatedAt | date-time (formatted string)| `strfmt.DateTime` |  | |  |  |
| ValidatorsID | string| `string` |  | |  |  |
| timestamp | [Timestamp](#timestamp)| `Timestamp` |  | |  |  |



### <span id="challenges"></span> Challenges


  

[][Challenge](#challenge)

### <span id="client"></span> Client


> Client - data structure that holds the client data
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| ID | string| `string` |  | |  |  |
| PublicKey | string| `string` |  | | The public key of the client |  |
| Version | string| `string` |  | | Version of the entity |  |
| creation_date | [Timestamp](#timestamp)| `Timestamp` |  | |  |  |



### <span id="coin"></span> Coin


> go:generate msgp -io=false -tests=false -v
  



| Name | Type | Go type | Default | Description | Example |
|------|------|---------| ------- |-------------|---------|
| Coin | uint64 (formatted integer)| uint64 | | go:generate msgp -io=false -tests=false -v |  |



### <span id="creation-date-field"></span> CreationDateField


> go:generate msgp -io=false -tests=false -v
CreationDateField - Can be used to add a creation date functionality to an entity */
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| creation_date | [Timestamp](#timestamp)| `Timestamp` |  | |  |  |



### <span id="d-k-g-key-share"></span> DKGKeyShare


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| ID | string| `string` |  | |  |  |
| Message | string| `string` |  | |  |  |
| Share | string| `string` |  | |  |  |
| Sign | string| `string` |  | |  |  |



### <span id="d-k-g-miner-nodes"></span> DKGMinerNodes


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| K | int64 (formatted integer)| `int64` |  | |  |  |
| KPercent | double (formatted number)| `float64` |  | |  |  |
| MaxN | int64 (formatted integer)| `int64` |  | |  |  |
| MinN | int64 (formatted integer)| `int64` |  | |  |  |
| N | int64 (formatted integer)| `int64` |  | |  |  |
| RevealedShares | map of int64 (formatted integer)| `map[string]int64` |  | |  |  |
| StartRound | int64 (formatted integer)| `int64` |  | | StartRound used to filter responses from old MB where sharders comes up. |  |
| T | int64 (formatted integer)| `int64` |  | |  |  |
| TPercent | double (formatted number)| `float64` |  | |  |  |
| Waited | map of boolean| `map[string]bool` |  | |  |  |
| XPercent | double (formatted number)| `float64` |  | |  |  |



### <span id="delegate-pool"></span> DelegatePool


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| DelegateID | string| `string` |  | |  |  |
| RoundCreated | int64 (formatted integer)| `int64` |  | |  |  |
| balance | [Coin](#coin)| `Coin` |  | |  |  |
| reward | [Coin](#coin)| `Coin` |  | |  |  |
| staked_at | [Timestamp](#timestamp)| `Timestamp` |  | |  |  |
| status | [PoolStatus](#pool-status)| `PoolStatus` |  | |  |  |



### <span id="delegate-pool-response"></span> DelegatePoolResponse


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| DelegateID | string| `string` |  | |  |  |
| RoundCreated | int64 (formatted integer)| `int64` |  | |  |  |
| RoundPoolLastUpdated | int64 (formatted integer)| `int64` |  | |  |  |
| balance | [Coin](#coin)| `Coin` |  | |  |  |
| reward | [Coin](#coin)| `Coin` |  | |  |  |
| staked_at | [Timestamp](#timestamp)| `Timestamp` |  | |  |  |
| status | [PoolStatus](#pool-status)| `PoolStatus` |  | |  |  |



### <span id="delegate-pool-stat"></span> DelegatePoolStat


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| DelegateID | string| `string` |  | |  |  |
| ID | string| `string` |  | |  |  |
| ProviderId | string| `string` |  | |  |  |
| RoundCreated | int64 (formatted integer)| `int64` |  | |  |  |
| Status | string| `string` |  | |  |  |
| UnStake | boolean| `bool` |  | |  |  |
| balance | [Coin](#coin)| `Coin` |  | |  |  |
| provider_type | [Provider](#provider)| `Provider` |  | |  |  |
| rewards | [Coin](#coin)| `Coin` |  | |  |  |
| staked_at | [Timestamp](#timestamp)| `Timestamp` |  | |  |  |
| total_penalty | [Coin](#coin)| `Coin` |  | |  |  |
| total_reward | [Coin](#coin)| `Coin` |  | |  |  |



### <span id="deleted-at"></span> DeletedAt


  


* composed type [NullTime](#null-time)

### <span id="duration"></span> Duration


> A Duration represents the elapsed time between two instants
as an int64 nanosecond count. The representation limits the
largest representable duration to approximately 290 years.
  



| Name | Type | Go type | Default | Description | Example |
|------|------|---------| ------- |-------------|---------|
| Duration | int64 (formatted integer)| int64 | | A Duration represents the elapsed time between two instants</br>as an int64 nanosecond count. The representation limits the</br>largest representable duration to approximately 290 years. |  |



### <span id="error"></span> Error


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| CreatedAt | date-time (formatted string)| `strfmt.DateTime` |  | |  |  |
| Error | string| `string` |  | |  |  |
| ID | uint64 (formatted integer)| `uint64` |  | |  |  |
| TransactionID | string| `string` |  | |  |  |



### <span id="event"></span> Event


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| BlockNumber | int64 (formatted integer)| `int64` |  | |  |  |
| CreatedAt | date-time (formatted string)| `strfmt.DateTime` |  | |  |  |
| Data | [interface{}](#interface)| `interface{}` |  | |  |  |
| ID | uint64 (formatted integer)| `uint64` |  | |  |  |
| Index | string| `string` |  | |  |  |
| TxHash | string| `string` |  | |  |  |
| tag | [EventTag](#event-tag)| `EventTag` |  | |  |  |
| type | [EventType](#event-type)| `EventType` |  | |  |  |



### <span id="event-tag"></span> EventTag


  

| Name | Type | Go type | Default | Description | Example |
|------|------|---------| ------- |-------------|---------|
| EventTag | int64 (formatted integer)| int64 | |  |  |



### <span id="event-type"></span> EventType


  

| Name | Type | Go type | Default | Description | Example |
|------|------|---------| ------- |-------------|---------|
| EventType | int64 (formatted integer)| int64 | |  |  |



### <span id="global-settings"></span> GlobalSettings


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| Fields | map of string| `map[string]string` |  | |  |  |
| Version | int64 (formatted integer)| `int64` |  | |  |  |



### <span id="group-shares-or-signs"></span> GroupSharesOrSigns


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| Shares | map of [ShareOrSigns](#share-or-signs)| `map[string]ShareOrSigns` |  | |  |  |



### <span id="hash-id-field"></span> HashIDField


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| Hash | string| `string` |  | |  |  |



### <span id="id-field"></span> IDField


> go:generate msgp -io=false -tests=false -v
IDField - Useful to embed this into all the entities and get consistent behavior */
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| ID | string| `string` |  | |  |  |



### <span id="immutable-model"></span> ImmutableModel


> type User struct {
model.Model
}
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| CreatedAt | date-time (formatted string)| `strfmt.DateTime` |  | |  |  |
| ID | uint64 (formatted integer)| `uint64` |  | |  |  |



### <span id="info"></span> Info


> Info - (informal) info of a node that can be shared with other nodes
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| AvgBlockTxns | int64 (formatted integer)| `int64` |  | |  |  |
| BuildTag | string| `string` |  | |  |  |
| StateMissingNodes | int64 (formatted integer)| `int64` |  | |  |  |
| miners_median_network_time | [Duration](#duration)| `Duration` |  | |  |  |



### <span id="int64-map"></span> Int64Map


  

[Int64Map](#int64-map)

### <span id="interface-map"></span> InterfaceMap


  

[InterfaceMap](#interface-map)

### <span id="key"></span> Key


> Key - a type for the merkle patricia trie node key
  



[]uint8 (formatted integer)

### <span id="m-p-k"></span> MPK


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| ID | string| `string` |  | |  |  |
| Mpk | []string| `[]string` |  | |  |  |



### <span id="magic-block"></span> MagicBlock


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| Hash | string| `string` |  | |  |  |
| K | int64 (formatted integer)| `int64` |  | |  |  |
| MagicBlockNumber | int64 (formatted integer)| `int64` |  | |  |  |
| N | int64 (formatted integer)| `int64` |  | |  |  |
| PreviousMagicBlockHash | string| `string` |  | |  |  |
| StartingRound | int64 (formatted integer)| `int64` |  | |  |  |
| T | int64 (formatted integer)| `int64` |  | |  |  |
| miners | [Pool](#pool)| `Pool` |  | |  |  |
| mpks | [Mpks](#mpks)| `Mpks` |  | |  |  |
| sharders | [Pool](#pool)| `Pool` |  | |  |  |
| share_or_signs | [GroupSharesOrSigns](#group-shares-or-signs)| `GroupSharesOrSigns` |  | |  |  |



### <span id="miner-aggregate"></span> MinerAggregate


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| BlocksFinalised | int64 (formatted integer)| `int64` |  | |  |  |
| CreatedAt | date-time (formatted string)| `strfmt.DateTime` |  | |  |  |
| ID | uint64 (formatted integer)| `uint64` |  | |  |  |
| IsKilled | boolean| `bool` |  | |  |  |
| IsShutdown | boolean| `bool` |  | |  |  |
| MinerID | string| `string` |  | |  |  |
| Round | int64 (formatted integer)| `int64` |  | |  |  |
| ServiceCharge | double (formatted number)| `float64` |  | |  |  |
| URL | string| `string` |  | |  |  |
| fees | [Coin](#coin)| `Coin` |  | |  |  |
| last_health_check | [Timestamp](#timestamp)| `Timestamp` |  | |  |  |
| total_rewards | [Coin](#coin)| `Coin` |  | |  |  |
| total_stake | [Coin](#coin)| `Coin` |  | |  |  |



### <span id="miner-node"></span> MinerNode


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| BuildTag | string| `string` |  | |  |  |
| Delete | boolean| `bool` |  | |  |  |
| HasBeenKilled | boolean| `bool` |  | |  |  |
| HasBeenKilled | boolean| `bool` |  | |  |  |
| HasBeenShutDown | boolean| `bool` |  | |  |  |
| Host | string| `string` |  | |  |  |
| ID | string| `string` |  | |  |  |
| LastSettingUpdateRound | int64 (formatted integer)| `int64` |  | | LastSettingUpdateRound will be set to round number when settings were updated |  |
| N2NHost | string| `string` |  | |  |  |
| Path | string| `string` |  | |  |  |
| Pools | map of [DelegatePool](#delegate-pool)| `map[string]DelegatePool` |  | |  |  |
| Port | int64 (formatted integer)| `int64` |  | |  |  |
| PublicKey | string| `string` |  | |  |  |
| ShortName | string| `string` |  | |  |  |
| last_health_check | [Timestamp](#timestamp)| `Timestamp` |  | |  |  |
| minter | [ApprovedMinter](#approved-minter)| `ApprovedMinter` |  | |  |  |
| node_type | [NodeType](#node-type)| `NodeType` |  | |  |  |
| provider_type | [Provider](#provider)| `Provider` |  | |  |  |
| rewards | [Coin](#coin)| `Coin` |  | |  |  |
| settings | [Settings](#settings)| `Settings` |  | |  |  |
| total_stake | [Coin](#coin)| `Coin` |  | |  |  |



### <span id="miner-nodes"></span> MinerNodes


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| Nodes | [][MinerNode](#miner-node)| `[]*MinerNode` |  | |  |  |



### <span id="miner-snapshot"></span> MinerSnapshot


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| BlocksFinalised | int64 (formatted integer)| `int64` |  | |  |  |
| CreationRound | int64 (formatted integer)| `int64` |  | |  |  |
| IsKilled | boolean| `bool` |  | |  |  |
| IsShutdown | boolean| `bool` |  | |  |  |
| MinerID | string| `string` |  | |  |  |
| Round | int64 (formatted integer)| `int64` |  | |  |  |
| ServiceCharge | double (formatted number)| `float64` |  | |  |  |
| fees | [Coin](#coin)| `Coin` |  | |  |  |
| total_rewards | [Coin](#coin)| `Coin` |  | |  |  |
| total_stake | [Coin](#coin)| `Coin` |  | |  |  |



### <span id="model"></span> Model


> type User struct {
gorm.Model
}
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| CreatedAt | date-time (formatted string)| `strfmt.DateTime` |  | |  |  |
| DeletedAt | [DeletedAt](#deleted-at)| `DeletedAt` |  | |  |  |
| ID | uint64 (formatted integer)| `uint64` |  | |  |  |
| UpdatedAt | date-time (formatted string)| `strfmt.DateTime` |  | |  |  |



### <span id="mpks"></span> Mpks


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| Mpks | map of [MPK](#m-p-k)| `map[string]MPK` |  | |  |  |



### <span id="n-o-id-field"></span> NOIDField


> NOIDFied - used when we just want to create a datastore entity that doesn't
have it's own id (like 1-to-many) that is only required to send it around with the parent key */
  



[interface{}](#interface)

### <span id="node"></span> Node


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| Description | string| `string` |  | |  |  |
| Host | string| `string` |  | |  |  |
| ID | string| `string` |  | |  |  |
| InPrevMB | boolean| `bool` |  | |  |  |
| N2NHost | string| `string` |  | |  |  |
| Path | string| `string` |  | |  |  |
| Port | int64 (formatted integer)| `int64` |  | |  |  |
| PublicKey | string| `string` |  | | The public key of the client |  |
| SetIndex | int64 (formatted integer)| `int64` |  | |  |  |
| Status | int64 (formatted integer)| `int64` |  | |  |  |
| Version | string| `string` |  | | Version of the entity |  |
| creation_date | [Timestamp](#timestamp)| `Timestamp` |  | |  |  |
| info | [Info](#info)| `Info` |  | |  |  |
| type | [NodeType](#node-type)| `NodeType` |  | |  |  |



### <span id="node-pool"></span> NodePool


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| DelegateID | string| `string` |  | |  |  |
| PoolID | string| `string` |  | |  |  |
| RoundCreated | int64 (formatted integer)| `int64` |  | |  |  |
| balance | [Coin](#coin)| `Coin` |  | |  |  |
| reward | [Coin](#coin)| `Coin` |  | |  |  |
| staked_at | [Timestamp](#timestamp)| `Timestamp` |  | |  |  |
| status | [PoolStatus](#pool-status)| `PoolStatus` |  | |  |  |



### <span id="node-response"></span> NodeResponse


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| BuildTag | string| `string` |  | |  |  |
| Delete | boolean| `bool` |  | |  |  |
| Host | string| `string` |  | |  |  |
| ID | string| `string` |  | |  |  |
| IsKilled | boolean| `bool` |  | |  |  |
| LastSettingUpdateRound | int64 (formatted integer)| `int64` |  | |  |  |
| N2NHost | string| `string` |  | |  |  |
| Path | string| `string` |  | |  |  |
| Pools | map of [DelegatePoolResponse](#delegate-pool-response)| `map[string]DelegatePoolResponse` |  | |  |  |
| Port | int64 (formatted integer)| `int64` |  | |  |  |
| PublicKey | string| `string` |  | |  |  |
| RoundServiceChargeLastUpdated | int64 (formatted integer)| `int64` |  | |  |  |
| ShortName | string| `string` |  | |  |  |
| last_health_check | [Timestamp](#timestamp)| `Timestamp` |  | |  |  |
| minter | [ApprovedMinter](#approved-minter)| `ApprovedMinter` |  | |  |  |
| node_type | [NodeType](#node-type)| `NodeType` |  | |  |  |
| rewards | [Coin](#coin)| `Coin` |  | |  |  |
| settings | [Settings](#settings)| `Settings` |  | |  |  |
| total_stake | [Coin](#coin)| `Coin` |  | |  |  |



### <span id="node-type"></span> NodeType


  

| Name | Type | Go type | Default | Description | Example |
|------|------|---------| ------- |-------------|---------|
| NodeType | int8 (formatted integer)| int8 | |  |  |



### <span id="null-time"></span> NullTime


> NullTime implements the [Scanner] interface so
it can be used as a scan destination, similar to [NullString].
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| Time | date-time (formatted string)| `strfmt.DateTime` |  | |  |  |
| Valid | boolean| `bool` |  | |  |  |



### <span id="phase"></span> Phase


  

| Name | Type | Go type | Default | Description | Example |
|------|------|---------| ------- |-------------|---------|
| Phase | int64 (formatted integer)| int64 | |  |  |



### <span id="phase-node"></span> PhaseNode


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| CurrentRound | int64 (formatted integer)| `int64` |  | |  |  |
| Restarts | int64 (formatted integer)| `int64` |  | |  |  |
| StartRound | int64 (formatted integer)| `int64` |  | |  |  |
| phase | [Phase](#phase)| `Phase` |  | |  |  |



### <span id="pool"></span> Pool


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| NodesMap | map of [Node](#node)| `map[string]Node` |  | |  |  |
| type | [NodeType](#node-type)| `NodeType` |  | |  |  |



### <span id="pool-member-info"></span> PoolMemberInfo


> PoolMemberInfo of a pool member
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| N2NHost | string| `string` |  | |  |  |
| Port | string| `string` |  | |  |  |
| PublicKey | string| `string` |  | |  |  |
| Type | string| `string` |  | |  |  |



### <span id="pool-members-info"></span> PoolMembersInfo


> PoolMembersInfo array of pool memebers
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| MembersInfo | [][PoolMemberInfo](#pool-member-info)| `[]*PoolMemberInfo` |  | |  |  |



### <span id="pool-status"></span> PoolStatus


  

| Name | Type | Go type | Default | Description | Example |
|------|------|---------| ------- |-------------|---------|
| PoolStatus | int64 (formatted integer)| int64 | |  |  |



### <span id="provider"></span> Provider


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| HasBeenKilled | boolean| `bool` |  | |  |  |
| HasBeenShutDown | boolean| `bool` |  | |  |  |
| ID | string| `string` |  | |  |  |
| last_health_check | [Timestamp](#timestamp)| `Timestamp` |  | |  |  |
| provider_type | [Provider](#provider)| `Provider` |  | |  |  |



### <span id="provider-rewards"></span> ProviderRewards


> ProviderRewards is a tables stores the rewards and total_rewards for all kinds of providers
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| CreatedAt | date-time (formatted string)| `strfmt.DateTime` |  | |  |  |
| ID | uint64 (formatted integer)| `uint64` |  | |  |  |
| ProviderID | string| `string` |  | |  |  |
| RoundServiceChargeLastUpdated | int64 (formatted integer)| `int64` |  | |  |  |
| UpdatedAt | date-time (formatted string)| `strfmt.DateTime` |  | |  |  |
| rewards | [Coin](#coin)| `Coin` |  | |  |  |
| total_rewards | [Coin](#coin)| `Coin` |  | |  |  |



### <span id="read-marker"></span> ReadMarker


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| Allocation | [Allocation](#allocation)| `Allocation` |  | |  |  |
| AllocationID | string| `string` |  | |  |  |
| AuthTicket | string| `string` |  | |  |  |
| BlobberID | string| `string` |  | |  |  |
| BlockNumber | int64 (formatted integer)| `int64` |  | |  |  |
| ClientID | string| `string` |  | |  |  |
| CreatedAt | date-time (formatted string)| `strfmt.DateTime` |  | |  |  |
| ID | uint64 (formatted integer)| `uint64` |  | |  |  |
| OwnerID | string| `string` |  | |  |  |
| PayerID | string| `string` |  | |  |  |
| ReadCounter | int64 (formatted integer)| `int64` |  | |  |  |
| ReadSize | double (formatted number)| `float64` |  | |  |  |
| Signature | string| `string` |  | |  |  |
| Timestamp | int64 (formatted integer)| `int64` |  | |  |  |
| TransactionID | string| `string` |  | |  |  |



### <span id="read-pool"></span> ReadPool


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| CreatedAt | date-time (formatted string)| `strfmt.DateTime` |  | |  |  |
| ID | uint64 (formatted integer)| `uint64` |  | |  |  |
| UpdatedAt | date-time (formatted string)| `strfmt.DateTime` |  | |  |  |
| UserID | string| `string` |  | |  |  |
| balance | [Coin](#coin)| `Coin` |  | |  |  |



### <span id="reward"></span> Reward


  

| Name | Type | Go type | Default | Description | Example |
|------|------|---------| ------- |-------------|---------|
| Reward | int64 (formatted integer)| int64 | |  |  |



### <span id="reward-delegate"></span> RewardDelegate


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| AllocationID | string| `string` |  | |  |  |
| BlockNumber | int64 (formatted integer)| `int64` |  | |  |  |
| CreatedAt | date-time (formatted string)| `strfmt.DateTime` |  | |  |  |
| ID | uint64 (formatted integer)| `uint64` |  | |  |  |
| PoolID | string| `string` |  | |  |  |
| ProviderID | string| `string` |  | |  |  |
| UpdatedAt | date-time (formatted string)| `strfmt.DateTime` |  | |  |  |
| amount | [Coin](#coin)| `Coin` |  | |  |  |
| reward_type | [Reward](#reward)| `Reward` |  | |  |  |



### <span id="reward-provider"></span> RewardProvider


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| AllocationID | string| `string` |  | |  |  |
| BlockNumber | int64 (formatted integer)| `int64` |  | |  |  |
| CreatedAt | date-time (formatted string)| `strfmt.DateTime` |  | |  |  |
| ID | uint64 (formatted integer)| `uint64` |  | |  |  |
| ProviderId | string| `string` |  | |  |  |
| UpdatedAt | date-time (formatted string)| `strfmt.DateTime` |  | |  |  |
| amount | [Coin](#coin)| `Coin` |  | |  |  |
| reward_type | [Reward](#reward)| `Reward` |  | |  |  |



### <span id="settings"></span> Settings


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| DelegateWallet | string| `string` |  | |  |  |
| MaxNumDelegates | int64 (formatted integer)| `int64` |  | |  |  |
| ServiceChargeRatio | double (formatted number)| `float64` |  | |  |  |
| min_stake | [Coin](#coin)| `Coin` |  | |  |  |



### <span id="sharder-aggregate"></span> SharderAggregate


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| CreatedAt | date-time (formatted string)| `strfmt.DateTime` |  | |  |  |
| ID | uint64 (formatted integer)| `uint64` |  | |  |  |
| IsKilled | boolean| `bool` |  | |  |  |
| IsShutdown | boolean| `bool` |  | |  |  |
| Round | int64 (formatted integer)| `int64` |  | |  |  |
| ServiceCharge | double (formatted number)| `float64` |  | |  |  |
| SharderID | string| `string` |  | |  |  |
| URL | string| `string` |  | |  |  |
| fees | [Coin](#coin)| `Coin` |  | |  |  |
| last_health_check | [Timestamp](#timestamp)| `Timestamp` |  | |  |  |
| total_rewards | [Coin](#coin)| `Coin` |  | |  |  |
| total_stake | [Coin](#coin)| `Coin` |  | |  |  |



### <span id="sharder-snapshot"></span> SharderSnapshot


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| CreationRound | int64 (formatted integer)| `int64` |  | |  |  |
| IsKilled | boolean| `bool` |  | |  |  |
| IsShutdown | boolean| `bool` |  | |  |  |
| Round | int64 (formatted integer)| `int64` |  | |  |  |
| ServiceCharge | double (formatted number)| `float64` |  | |  |  |
| SharderID | string| `string` |  | |  |  |
| fees | [Coin](#coin)| `Coin` |  | |  |  |
| total_rewards | [Coin](#coin)| `Coin` |  | |  |  |
| total_stake | [Coin](#coin)| `Coin` |  | |  |  |



### <span id="share-or-signs"></span> ShareOrSigns


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| ID | string| `string` |  | |  |  |
| ShareOrSigns | map of [DKGKeyShare](#d-k-g-key-share)| `map[string]DKGKeyShare` |  | |  |  |



### <span id="simple-node"></span> SimpleNode


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| BuildTag | string| `string` |  | |  |  |
| Delete | boolean| `bool` |  | |  |  |
| HasBeenKilled | boolean| `bool` |  | |  |  |
| HasBeenShutDown | boolean| `bool` |  | |  |  |
| Host | string| `string` |  | |  |  |
| ID | string| `string` |  | |  |  |
| LastSettingUpdateRound | int64 (formatted integer)| `int64` |  | | LastSettingUpdateRound will be set to round number when settings were updated |  |
| N2NHost | string| `string` |  | |  |  |
| Path | string| `string` |  | |  |  |
| Port | int64 (formatted integer)| `int64` |  | |  |  |
| PublicKey | string| `string` |  | |  |  |
| ShortName | string| `string` |  | |  |  |
| last_health_check | [Timestamp](#timestamp)| `Timestamp` |  | |  |  |
| node_type | [NodeType](#node-type)| `NodeType` |  | |  |  |
| provider_type | [Provider](#provider)| `Provider` |  | |  |  |
| total_stake | [Coin](#coin)| `Coin` |  | |  |  |



### <span id="simple-node-response"></span> SimpleNodeResponse


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| BuildTag | string| `string` |  | |  |  |
| Delete | boolean| `bool` |  | |  |  |
| Host | string| `string` |  | |  |  |
| ID | string| `string` |  | |  |  |
| IsKilled | boolean| `bool` |  | |  |  |
| LastSettingUpdateRound | int64 (formatted integer)| `int64` |  | |  |  |
| N2NHost | string| `string` |  | |  |  |
| Path | string| `string` |  | |  |  |
| Port | int64 (formatted integer)| `int64` |  | |  |  |
| PublicKey | string| `string` |  | |  |  |
| RoundServiceChargeLastUpdated | int64 (formatted integer)| `int64` |  | |  |  |
| ShortName | string| `string` |  | |  |  |
| last_health_check | [Timestamp](#timestamp)| `Timestamp` |  | |  |  |
| node_type | [NodeType](#node-type)| `NodeType` |  | |  |  |
| total_stake | [Coin](#coin)| `Coin` |  | |  |  |



### <span id="simple-nodes"></span> SimpleNodes


> not thread safe
  



[SimpleNodes](#simple-nodes)

### <span id="snapshot"></span> Snapshot


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| ActiveAllocatedDelta | int64 (formatted integer)| `int64` |  | |  |  |
| AllocatedStorage | int64 (formatted integer)| `int64` |  | |  |  |
| AuthorizerCount | int64 (formatted integer)| `int64` |  | |  |  |
| BlobberCount | int64 (formatted integer)| `int64` |  | |  |  |
| BlobberTotalRewards | int64 (formatted integer)| `int64` |  | |  |  |
| BlockCount | int64 (formatted integer)| `int64` |  | |  |  |
| ClientLocks | int64 (formatted integer)| `int64` |  | |  |  |
| CreatedAt | int64 (formatted integer)| `int64` |  | |  |  |
| MaxCapacityStorage | int64 (formatted integer)| `int64` |  | |  |  |
| MinerCount | int64 (formatted integer)| `int64` |  | |  |  |
| MinerTotalRewards | int64 (formatted integer)| `int64` |  | |  |  |
| Round | int64 (formatted integer)| `int64` |  | |  |  |
| SharderCount | int64 (formatted integer)| `int64` |  | |  |  |
| SharderTotalRewards | int64 (formatted integer)| `int64` |  | |  |  |
| StakedStorage | int64 (formatted integer)| `int64` |  | |  |  |
| StorageTokenStake | int64 (formatted integer)| `int64` |  | |  |  |
| SuccessfulChallenges | int64 (formatted integer)| `int64` |  | |  |  |
| TotalAllocations | int64 (formatted integer)| `int64` |  | |  |  |
| TotalChallengePools | int64 (formatted integer)| `int64` |  | |  |  |
| TotalChallenges | int64 (formatted integer)| `int64` |  | |  |  |
| TotalMint | int64 (formatted integer)| `int64` |  | |  |  |
| TotalReadPoolLocked | int64 (formatted integer)| `int64` |  | |  |  |
| TotalRewards | int64 (formatted integer)| `int64` |  | |  |  |
| TotalStaked | int64 (formatted integer)| `int64` |  | | updated from blobber snapshot aggregate table |  |
| TotalTxnFee | int64 (formatted integer)| `int64` |  | |  |  |
| TransactionsCount | int64 (formatted integer)| `int64` |  | |  |  |
| UniqueAddresses | int64 (formatted integer)| `int64` |  | |  |  |
| UsedStorage | int64 (formatted integer)| `int64` |  | |  |  |
| ValidatorCount | int64 (formatted integer)| `int64` |  | |  |  |
| ZCNSupply | int64 (formatted integer)| `int64` |  | |  |  |



### <span id="stake-pool"></span> StakePool


> StakePool holds delegate information for an 0chain providers
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| HasBeenKilled | boolean| `bool` |  | |  |  |
| Pools | map of [DelegatePool](#delegate-pool)| `map[string]DelegatePool` |  | |  |  |
| minter | [ApprovedMinter](#approved-minter)| `ApprovedMinter` |  | |  |  |
| rewards | [Coin](#coin)| `Coin` |  | |  |  |
| settings | [Settings](#settings)| `Settings` |  | |  |  |



### <span id="stake-pool-response"></span> StakePoolResponse


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| Pools | map of [DelegatePoolResponse](#delegate-pool-response)| `map[string]DelegatePoolResponse` |  | |  |  |
| minter | [ApprovedMinter](#approved-minter)| `ApprovedMinter` |  | |  |  |
| rewards | [Coin](#coin)| `Coin` |  | |  |  |
| settings | [Settings](#settings)| `Settings` |  | |  |  |



### <span id="stake-pool-stat"></span> StakePoolStat


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| Delegate | [][DelegatePoolStat](#delegate-pool-stat)| `[]*DelegatePoolStat` |  | |  |  |
| ID | string| `string` |  | |  |  |
| balance | [Coin](#coin)| `Coin` |  | |  |  |
| penalty | [Coin](#coin)| `Coin` |  | |  |  |
| rewards | [Coin](#coin)| `Coin` |  | |  |  |
| settings | [Settings](#settings)| `Settings` |  | |  |  |
| stake_total | [Coin](#coin)| `Coin` |  | |  |  |
| total_rewards | [Coin](#coin)| `Coin` |  | |  |  |



### <span id="state"></span> State


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| Nonce | int64 (formatted integer)| `int64` |  | | Latest nonce used by the client wallet. |  |
| Round | int64 (formatted integer)| `int64` |  | | Latest round when the latest txn happened. |  |
| TxnHash | string| `string` |  | | Latest transaction run by the client wallet. |  |
| balance | [Coin](#coin)| `Coin` |  | |  |  |



### <span id="string-map"></span> StringMap


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| Fields | map of string| `map[string]string` |  | |  |  |



### <span id="timestamp"></span> Timestamp


> go:generate msgp -io=false -tests=false -v
Timestamp - just a wrapper to control the json encoding */
  



| Name | Type | Go type | Default | Description | Example |
|------|------|---------| ------- |-------------|---------|
| Timestamp | int64 (formatted integer)| int64 | | go:generate msgp -io=false -tests=false -v</br>Timestamp - just a wrapper to control the json encoding */ |  |



### <span id="transaction"></span> Transaction


> Transaction model to save the transaction data
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| ChainID | string| `string` | ✓ | | ChainID - the chain id of the transaction |  |
| ClientID | string| `string` | ✓ | | ClientID of the client issuing the transaction |  |
| Hash | string| `string` |  | |  |  |
| Nonce | int64 (formatted integer)| `int64` | ✓ | | Nonce - the nonce associated with the transaction |  |
| OutputHash | string| `string` | ✓ | | OutputHash - the hash of the transaction output |  |
| PublicKey | string| `string` | ✓ | | Public key of the client issuing the transaction |  |
| Signature | string| `string` | ✓ | | Signature - Issuer signature of the transaction |  |
| Status | int64 (formatted integer)| `int64` | ✓ | | Status - the status of the transaction |  |
| ToClientID | string| `string` | ✓ | | ToClientID - the client id of the recipient, the other party in the transaction. It can be a client id or the address of a smart contract |  |
| TransactionData | string| `string` | ✓ | | TransactionData - the data associated with the transaction |  |
| TransactionOutput | string| `string` | ✓ | | TransactionOutput - the output of the transaction |  |
| TransactionType | int64 (formatted integer)| `int64` | ✓ | | TransactionType - the type of the transaction. </br>Possible values are:</br>0: TxnTypeSend - A transaction to send tokens to another account, state is maintained by account.</br>10: TxnTypeData - A transaction to just store a piece of data on the block chain.</br>1000: TxnTypeSmartContract - A smart contract transaction type. |  |
| Version | string| `string` |  | | Version of the entity |  |
| creation_date | [Timestamp](#timestamp)| `Timestamp` | ✓ | |  |  |
| transaction_fee | [Coin](#coin)| `Coin` | ✓ | |  |  |
| transaction_value | [Coin](#coin)| `Coin` | ✓ | |  |  |



### <span id="unverified-block-body"></span> UnverifiedBlockBody


> UnverifiedBlockBody - used to compute the signature
This is what is used to verify the correctness of the block & the associated signature
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| LatestFinalizedMagicBlockHash | string| `string` |  | |  |  |
| LatestFinalizedMagicBlockRound | int64 (formatted integer)| `int64` |  | |  |  |
| MinerID | string| `string` |  | |  |  |
| PrevBlockVerificationTickets | [][VerificationTicket](#verification-ticket)| `[]*VerificationTicket` |  | |  |  |
| PrevHash | string| `string` |  | |  |  |
| Round | int64 (formatted integer)| `int64` |  | |  |  |
| RoundRandomSeed | int64 (formatted integer)| `int64` |  | |  |  |
| RoundTimeoutCount | int64 (formatted integer)| `int64` |  | |  |  |
| Txns | [][Transaction](#transaction)| `[]*Transaction` |  | | The entire transaction payload to represent full block |  |
| Version | string| `string` |  | | Version of the entity |  |
| creation_date | [Timestamp](#timestamp)| `Timestamp` |  | |  |  |
| state_hash | [Key](#key)| `Key` |  | |  |  |



### <span id="updatable-model"></span> UpdatableModel


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| CreatedAt | date-time (formatted string)| `strfmt.DateTime` |  | |  |  |
| ID | uint64 (formatted integer)| `uint64` |  | |  |  |
| UpdatedAt | date-time (formatted string)| `strfmt.DateTime` |  | |  |  |



### <span id="user"></span> User


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| CreatedAt | date-time (formatted string)| `strfmt.DateTime` |  | |  |  |
| ID | uint64 (formatted integer)| `uint64` |  | |  |  |
| MintNonce | int64 (formatted integer)| `int64` |  | |  |  |
| Nonce | int64 (formatted integer)| `int64` |  | |  |  |
| Round | int64 (formatted integer)| `int64` |  | |  |  |
| TxnHash | string| `string` |  | |  |  |
| UpdatedAt | date-time (formatted string)| `strfmt.DateTime` |  | |  |  |
| UserID | string| `string` |  | |  |  |
| balance | [Coin](#coin)| `Coin` |  | |  |  |



### <span id="user-aggregate"></span> UserAggregate


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| CollectedReward | int64 (formatted integer)| `int64` |  | |  |  |
| CreatedAt | date-time (formatted string)| `strfmt.DateTime` |  | |  |  |
| PayedFees | int64 (formatted integer)| `int64` |  | |  |  |
| ReadPoolTotal | int64 (formatted integer)| `int64` |  | |  |  |
| Round | int64 (formatted integer)| `int64` |  | |  |  |
| TotalReward | int64 (formatted integer)| `int64` |  | |  |  |
| TotalStake | int64 (formatted integer)| `int64` |  | |  |  |
| UserID | string| `string` |  | |  |  |
| WritePoolTotal | int64 (formatted integer)| `int64` |  | |  |  |



### <span id="user-pool-stat"></span> UserPoolStat


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| Pools | map of [[]*DelegatePoolStat](#delegate-pool-stat)| `map[string][]DelegatePoolStat` |  | |  |  |



### <span id="validator"></span> Validator


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| BaseUrl | string| `string` |  | |  |  |
| CreatedAt | date-time (formatted string)| `strfmt.DateTime` |  | |  |  |
| CreationRound | int64 (formatted integer)| `int64` |  | |  |  |
| DelegateWallet | string| `string` |  | |  |  |
| Downtime | uint64 (formatted integer)| `uint64` |  | |  |  |
| ID | string| `string` |  | |  |  |
| IsKilled | boolean| `bool` |  | |  |  |
| IsShutdown | boolean| `bool` |  | |  |  |
| NumDelegates | int64 (formatted integer)| `int64` |  | |  |  |
| PublicKey | string| `string` |  | |  |  |
| ServiceCharge | double (formatted number)| `float64` |  | |  |  |
| UpdatedAt | date-time (formatted string)| `strfmt.DateTime` |  | |  |  |
| last_health_check | [Timestamp](#timestamp)| `Timestamp` |  | |  |  |
| rewards | [ProviderRewards](#provider-rewards)| `ProviderRewards` |  | |  |  |
| total_stake | [Coin](#coin)| `Coin` |  | |  |  |



### <span id="validator-aggregate"></span> ValidatorAggregate


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| CreatedAt | date-time (formatted string)| `strfmt.DateTime` |  | |  |  |
| ID | uint64 (formatted integer)| `uint64` |  | |  |  |
| IsKilled | boolean| `bool` |  | |  |  |
| IsShutdown | boolean| `bool` |  | |  |  |
| Round | int64 (formatted integer)| `int64` |  | |  |  |
| ServiceCharge | double (formatted number)| `float64` |  | |  |  |
| URL | string| `string` |  | |  |  |
| ValidatorID | string| `string` |  | |  |  |
| last_health_check | [Timestamp](#timestamp)| `Timestamp` |  | |  |  |
| total_rewards | [Coin](#coin)| `Coin` |  | |  |  |
| total_stake | [Coin](#coin)| `Coin` |  | |  |  |



### <span id="validator-snapshot"></span> ValidatorSnapshot


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| CreationRound | int64 (formatted integer)| `int64` |  | |  |  |
| IsKilled | boolean| `bool` |  | |  |  |
| IsShutdown | boolean| `bool` |  | |  |  |
| Round | int64 (formatted integer)| `int64` |  | |  |  |
| ServiceCharge | double (formatted number)| `float64` |  | |  |  |
| ValidatorID | string| `string` |  | |  |  |
| total_rewards | [Coin](#coin)| `Coin` |  | |  |  |
| total_stake | [Coin](#coin)| `Coin` |  | |  |  |



### <span id="verification-ticket"></span> VerificationTicket


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| Signature | string| `string` |  | |  |  |
| VerifierID | string| `string` |  | |  |  |



### <span id="version-field"></span> VersionField


> go:generate msgp -io=false -tests=false -v
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| Version | string| `string` |  | | Version of the entity |  |



### <span id="write-marker"></span> WriteMarker


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| Allocation | [Allocation](#allocation)| `Allocation` |  | |  |  |
| AllocationID | string| `string` |  | |  |  |
| AllocationRoot | string| `string` |  | |  |  |
| BlobberID | string| `string` |  | |  |  |
| BlockNumber | int64 (formatted integer)| `int64` |  | |  |  |
| ChainHash | string| `string` |  | |  |  |
| ChainSize | int64 (formatted integer)| `int64` |  | |  |  |
| ClientID | string| `string` |  | |  |  |
| CreatedAt | date-time (formatted string)| `strfmt.DateTime` |  | |  |  |
| FileMetaRoot | string| `string` |  | |  |  |
| ID | uint64 (formatted integer)| `uint64` |  | |  |  |
| PreviousAllocationRoot | string| `string` |  | |  |  |
| Signature | string| `string` |  | |  |  |
| Size | int64 (formatted integer)| `int64` |  | |  |  |
| Timestamp | int64 (formatted integer)| `int64` |  | |  |  |
| TransactionID | string| `string` |  | |  |  |
| UpdatedAt | date-time (formatted string)| `strfmt.DateTime` |  | |  |  |



### <span id="event-list"></span> eventList


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| Events | [][Event](#event)| `[]*Event` |  | |  |  |



### <span id="node-stat"></span> nodeStat


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| BuildTag | string| `string` |  | |  |  |
| Delete | boolean| `bool` |  | |  |  |
| Host | string| `string` |  | |  |  |
| ID | string| `string` |  | |  |  |
| IsKilled | boolean| `bool` |  | |  |  |
| LastSettingUpdateRound | int64 (formatted integer)| `int64` |  | |  |  |
| N2NHost | string| `string` |  | |  |  |
| Path | string| `string` |  | |  |  |
| Pools | map of [DelegatePoolResponse](#delegate-pool-response)| `map[string]DelegatePoolResponse` |  | |  |  |
| Port | int64 (formatted integer)| `int64` |  | |  |  |
| PublicKey | string| `string` |  | |  |  |
| RoundServiceChargeLastUpdated | int64 (formatted integer)| `int64` |  | |  |  |
| ShortName | string| `string` |  | |  |  |
| TotalReward | int64 (formatted integer)| `int64` |  | |  |  |
| last_health_check | [Timestamp](#timestamp)| `Timestamp` |  | |  |  |
| minter | [ApprovedMinter](#approved-minter)| `ApprovedMinter` |  | |  |  |
| node_type | [NodeType](#node-type)| `NodeType` |  | |  |  |
| rewards | [Coin](#coin)| `Coin` |  | |  |  |
| settings | [Settings](#settings)| `Settings` |  | |  |  |
| total_stake | [Coin](#coin)| `Coin` |  | |  |  |


