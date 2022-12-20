


# 0chain Api:
  

## Informations

### Version

0.0.1

## Content negotiation

### URI Schemes
  * http
  * https

### Consumes
  * application/json

### Produces
  * application/json

## All endpoints

###  operations

| Method  | URI     | Name   | Summary |
|---------|---------|--------|---------|
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d3/GetGlobalConfig | [get global config](#get-global-config) |  |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/allocWrittenSizePerPeriod | [alloc written size per period](#alloc-written-size-per-period) |  |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/alloc_blobbers | [alloc blobbers](#alloc-blobbers) | returns list of all blobbers alive that match the allocation request. |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/alloc_read_size | [alloc read size](#alloc-read-size) |  |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/alloc_write_marker_count | [alloc write marker count](#alloc-write-marker-count) |  |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/alloc_written_size | [alloc written size](#alloc-written-size) |  |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/allocation | [allocation](#allocation) |  |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/allocation_min_lock | [allocation min lock](#allocation-min-lock) | Calculates the cost of a new allocation request. |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/allocations | [allocations](#allocations) |  |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/average-write-price | [average write price](#average-write-price) |  |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/blobber-challenges | [blobber challenges](#blobber-challenges) |  |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/blobber-rank | [blobber rank](#blobber-rank) | Gets the rank of a blobber. |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/blobber_ids | [blobber ids](#blobber-ids) |  |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/blobbers-by-geolocation | [blobbers by geolocation](#blobbers-by-geolocation) | Returns a list of all blobbers within a rectangle defined by maximum and minimum latitude and longitude values. |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/blobbers-by-rank | [blobbers by rank](#blobbers-by-rank) |  |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/block | [block](#block) |  |
| GET | /v1/chain/get/stats | [chainstatus](#chainstatus) |  |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/collected_reward | [collected reward](#collected-reward) | Returns collected reward for a client_id.
> Note: start-date and end-date resolves to the closest block number for those timestamps on the network. |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/configs | [configs](#configs) |  |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/count_readmarkers | [count readmarkers](#count-readmarkers) |  |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/delegate-rewards | [delegate rewards](#delegate-rewards) |  |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/errors | [errors](#errors) |  |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d3/faucet_config | [faucet config](#faucet-config) |  |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/free_alloc_blobbers | [free alloc blobbers](#free-alloc-blobbers) | returns list of all blobbers alive that match the free allocation request. |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/alloc-blobber-term | [get alloc blobber terms](#get-alloc-blobber-terms) |  |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d3/getAuthorizer | [get authorizer](#get-authorizer) |  |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712e0/getAuthorizerNodes | [get authorizer nodes](#get-authorizer-nodes) |  |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/getBlobber | [get blobber](#get-blobber) |  |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/getchallenge | [get challenge](#get-challenge) |  |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/getChallengePoolStat | [get challenge pool stat](#get-challenge-pool-stat) |  |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/getClientPools | [get client pools](#get-client-pools) |  |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/getDkgList | [get dkg list](#get-dkg-list) |  |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/getEvents | [get events](#get-events) |  |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/getGroupShareOrSigns | [get group share or signs](#get-group-share-or-signs) |  |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/getStakePoolStat | [get m s stake pool stat](#get-m-s-stake-pool-stat) |  |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/getMagicBlock | [get magic block](#get-magic-block) |  |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/getMinerList | [get miner list](#get-miner-list) |  |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/getMpksList | [get mpks list](#get-mpks-list) |  |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/getNodepool | [get nodepool](#get-nodepool) |  |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/getPhase | [get phase](#get-phase) |  |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/getPoolInfo | [get pool info](#get-pool-info) |  |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/getReadPoolStat | [get read pool stat](#get-read-pool-stat) |  |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/getSharderKeepList | [get sharder keep list](#get-sharder-keep-list) |  |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/getSharderList | [get sharder list](#get-sharder-list) |  |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/getStakePoolStat | [get stake pool stat](#get-stake-pool-stat) |  |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/getUserLockedTotal | [get user locked total](#get-user-locked-total) |  |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/getUserPools | [get user pools](#get-user-pools) |  |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/getUserStakePoolStat | [get user stake pool stat](#get-user-stake-pool-stat) |  |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/getWriteMarkers | [get write markers](#get-write-markers) |  |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/get_blobber_total_stakes | [get blobber total stakes](#get-blobber-total-stakes) |  |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/get_blocks | [get blocks](#get-blocks) | Gets block information for all blocks. Todo: We need to add a filter to this. |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/get_miner_geolocations | [get miner geolocations](#get-miner-geolocations) |  |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/get_miners_stake | [get miners stake](#get-miners-stake) |  |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/get_miners_stats | [get miners stats](#get-miners-stats) |  |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/get_sharder_geolocations | [get sharder geolocations](#get-sharder-geolocations) |  |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/get_sharders_stake | [get sharders stake](#get-sharders-stake) |  |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/get_sharders_stats | [get sharders stats](#get-sharders-stats) |  |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/get_validator | [get validator](#get-validator) |  |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/getblobbers | [getblobbers](#getblobbers) | Gets list of all blobbers alive (e.g. excluding blobbers with zero capacity). |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d3/globalPeriodicLimit | [global periodic limit](#global-periodic-limit) |  |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/globalSettings | [global settings](#global-settings) |  |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/latestreadmarker | [latestreadmarker](#latestreadmarker) |  |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/nodePoolStat | [node pool stat](#node-pool-stat) |  |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/nodeStat | [node stat](#node-stat) |  |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/openchallenges | [openchallenges](#openchallenges) |  |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d3/personalPeriodicLimit | [personal periodic limit](#personal-periodic-limit) |  |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d3/pourAmount | [pour amount](#pour-amount) |  |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/provider-rewards | [provider rewards](#provider-rewards) |  |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/readmarkers | [readmarkers](#readmarkers) |  |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/replicate-blobber-aggregate | [replicate blobber aggregates](#replicate-blobber-aggregates) |  |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/replicate-snapshots | [replicate snapshots](#replicate-snapshots) |  |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/search | [search](#search) |  |
| GET | /v1/sharder/get/stats | [sharderstats](#sharderstats) |  |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/storage-config | [storage config](#storage-config) |  |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/timestamp-to-round | [timestamps to rounds](#timestamps-to-rounds) |  |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/total-stored-data | [total stored data](#total-stored-data) | Gets the total data currently storage used across all blobbers. |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/transaction | [transaction](#transaction) |  |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/transactionHashes | [transaction hashes](#transaction-hashes) |  |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/transactions | [transactions](#transactions) |  |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/vesting_config | [vesting config](#vesting-config) |  |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/writemarkers | [writemarkers](#writemarkers) |  |
  


## Paths

### <span id="get-global-config"></span> get global config (*GetGlobalConfig*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d3/GetGlobalConfig
```

get zcn configuration settings

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#get-global-config-200) | OK | StringMap |  | [schema](#get-global-config-200-schema) |
| [404](#get-global-config-404) | Not Found |  |  | [schema](#get-global-config-404-schema) |

#### Responses


##### <span id="get-global-config-200"></span> 200 - StringMap
Status: OK

###### <span id="get-global-config-200-schema"></span> Schema
   
  

[StringMap](#string-map)

##### <span id="get-global-config-404"></span> 404
Status: Not Found

###### <span id="get-global-config-404-schema"></span> Schema

### <span id="alloc-written-size-per-period"></span> alloc written size per period (*allocWrittenSizePerPeriod*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/allocWrittenSizePerPeriod
```

Total amount of data added during given blocks

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| block-end | `query` | string | `string` |  | ✓ |  | end block number |
| block-start | `query` | string | `string` |  | ✓ |  | start block number |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#alloc-written-size-per-period-200) | OK | Int64Map |  | [schema](#alloc-written-size-per-period-200-schema) |
| [400](#alloc-written-size-per-period-400) | Bad Request |  |  | [schema](#alloc-written-size-per-period-400-schema) |

#### Responses


##### <span id="alloc-written-size-per-period-200"></span> 200 - Int64Map
Status: OK

###### <span id="alloc-written-size-per-period-200-schema"></span> Schema
   
  

[Int64Map](#int64-map)

##### <span id="alloc-written-size-per-period-400"></span> 400
Status: Bad Request

###### <span id="alloc-written-size-per-period-400-schema"></span> Schema

### <span id="alloc-blobbers"></span> returns list of all blobbers alive that match the allocation request. (*alloc_blobbers*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/alloc_blobbers
```

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| allocation_data | `query` | string | `string` |  | ✓ |  | allocation data |
| limit | `query` | string | `string` |  |  |  | limit |
| offset | `query` | string | `string` |  |  |  | offset |
| sort | `query` | string | `string` |  |  |  | desc or asc |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#alloc-blobbers-200) | OK |  |  | [schema](#alloc-blobbers-200-schema) |
| [400](#alloc-blobbers-400) | Bad Request |  |  | [schema](#alloc-blobbers-400-schema) |

#### Responses


##### <span id="alloc-blobbers-200"></span> 200
Status: OK

###### <span id="alloc-blobbers-200-schema"></span> Schema

##### <span id="alloc-blobbers-400"></span> 400
Status: Bad Request

###### <span id="alloc-blobbers-400-schema"></span> Schema

### <span id="alloc-read-size"></span> alloc read size (*alloc_read_size*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/alloc_read_size
```

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| allocation_id | `query` | string | `string` |  | ✓ |  | allocation for which to get challenge pools statistics |
| block_number | `query` | string | `string` |  | ✓ |  | block number |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#alloc-read-size-200) | OK | challengePoolStat |  | [schema](#alloc-read-size-200-schema) |
| [400](#alloc-read-size-400) | Bad Request |  |  | [schema](#alloc-read-size-400-schema) |

#### Responses


##### <span id="alloc-read-size-200"></span> 200 - challengePoolStat
Status: OK

###### <span id="alloc-read-size-200-schema"></span> Schema
   
  

[ChallengePoolStat](#challenge-pool-stat)

##### <span id="alloc-read-size-400"></span> 400
Status: Bad Request

###### <span id="alloc-read-size-400-schema"></span> Schema

### <span id="alloc-write-marker-count"></span> alloc write marker count (*alloc_write_marker_count*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/alloc_write_marker_count
```

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| allocation_id | `query` | string | `string` |  | ✓ |  | allocation for which to get challenge pools statistics |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#alloc-write-marker-count-200) | OK | challengePoolStat |  | [schema](#alloc-write-marker-count-200-schema) |
| [400](#alloc-write-marker-count-400) | Bad Request |  |  | [schema](#alloc-write-marker-count-400-schema) |

#### Responses


##### <span id="alloc-write-marker-count-200"></span> 200 - challengePoolStat
Status: OK

###### <span id="alloc-write-marker-count-200-schema"></span> Schema
   
  

[ChallengePoolStat](#challenge-pool-stat)

##### <span id="alloc-write-marker-count-400"></span> 400
Status: Bad Request

###### <span id="alloc-write-marker-count-400-schema"></span> Schema

### <span id="alloc-written-size"></span> alloc written size (*alloc_written_size*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/alloc_written_size
```

statistic for all locked tokens of a challenge pool

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| allocation_id | `query` | string | `string` |  | ✓ |  | allocation for which to get challenge pools statistics |
| block_number | `query` | string | `string` |  | ✓ |  | block number |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#alloc-written-size-200) | OK | challengePoolStat |  | [schema](#alloc-written-size-200-schema) |
| [400](#alloc-written-size-400) | Bad Request |  |  | [schema](#alloc-written-size-400-schema) |

#### Responses


##### <span id="alloc-written-size-200"></span> 200 - challengePoolStat
Status: OK

###### <span id="alloc-written-size-200-schema"></span> Schema
   
  

[ChallengePoolStat](#challenge-pool-stat)

##### <span id="alloc-written-size-400"></span> 400
Status: Bad Request

###### <span id="alloc-written-size-400-schema"></span> Schema

### <span id="allocation"></span> allocation (*allocation*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/allocation
```

Gets allocation object

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| allocation | `query` | string | `string` |  | ✓ |  | offset |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#allocation-200) | OK | StorageAllocation |  | [schema](#allocation-200-schema) |
| [400](#allocation-400) | Bad Request |  |  | [schema](#allocation-400-schema) |
| [500](#allocation-500) | Internal Server Error |  |  | [schema](#allocation-500-schema) |

#### Responses


##### <span id="allocation-200"></span> 200 - StorageAllocation
Status: OK

###### <span id="allocation-200-schema"></span> Schema
   
  

[StorageAllocation](#storage-allocation)

##### <span id="allocation-400"></span> 400
Status: Bad Request

###### <span id="allocation-400-schema"></span> Schema

##### <span id="allocation-500"></span> 500
Status: Internal Server Error

###### <span id="allocation-500-schema"></span> Schema

### <span id="allocation-min-lock"></span> Calculates the cost of a new allocation request. (*allocation_min_lock*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/allocation_min_lock
```

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| allocation_data | `query` | string | `string` |  | ✓ |  | json marshall of new allocation request input data |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#allocation-min-lock-200) | OK | Int64Map |  | [schema](#allocation-min-lock-200-schema) |
| [400](#allocation-min-lock-400) | Bad Request |  |  | [schema](#allocation-min-lock-400-schema) |
| [500](#allocation-min-lock-500) | Internal Server Error |  |  | [schema](#allocation-min-lock-500-schema) |

#### Responses


##### <span id="allocation-min-lock-200"></span> 200 - Int64Map
Status: OK

###### <span id="allocation-min-lock-200-schema"></span> Schema
   
  

[Int64Map](#int64-map)

##### <span id="allocation-min-lock-400"></span> 400
Status: Bad Request

###### <span id="allocation-min-lock-400-schema"></span> Schema

##### <span id="allocation-min-lock-500"></span> 500
Status: Internal Server Error

###### <span id="allocation-min-lock-500-schema"></span> Schema

### <span id="allocations"></span> allocations (*allocations*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/allocations
```

Gets a list of allocation information for allocations owned by the client

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| client | `query` | string | `string` |  | ✓ |  | owner of allocations we wish to list |
| limit | `query` | string | `string` |  |  |  | limit |
| offset | `query` | string | `string` |  |  |  | offset |
| sort | `query` | string | `string` |  |  |  | desc or asc |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#allocations-200) | OK | StorageAllocation |  | [schema](#allocations-200-schema) |
| [400](#allocations-400) | Bad Request |  |  | [schema](#allocations-400-schema) |
| [500](#allocations-500) | Internal Server Error |  |  | [schema](#allocations-500-schema) |

#### Responses


##### <span id="allocations-200"></span> 200 - StorageAllocation
Status: OK

###### <span id="allocations-200-schema"></span> Schema
   
  

[][StorageAllocation](#storage-allocation)

##### <span id="allocations-400"></span> 400
Status: Bad Request

###### <span id="allocations-400-schema"></span> Schema

##### <span id="allocations-500"></span> 500
Status: Internal Server Error

###### <span id="allocations-500-schema"></span> Schema

### <span id="average-write-price"></span> average write price (*average-write-price*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/average-write-price
```

Gets the average write price across all blobbers

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#average-write-price-200) | OK | Int64Map |  | [schema](#average-write-price-200-schema) |
| [400](#average-write-price-400) | Bad Request |  |  | [schema](#average-write-price-400-schema) |

#### Responses


##### <span id="average-write-price-200"></span> 200 - Int64Map
Status: OK

###### <span id="average-write-price-200-schema"></span> Schema
   
  

[Int64Map](#int64-map)

##### <span id="average-write-price-400"></span> 400
Status: Bad Request

###### <span id="average-write-price-400-schema"></span> Schema

### <span id="blobber-challenges"></span> blobber challenges (*blobber-challenges*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/blobber-challenges
```

Gets challenges for a blobber by challenge id

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| end | `query` | string | `string` |  | ✓ |  | end time of interval |
| id | `query` | string | `string` |  | ✓ |  | id of blobber |
| start | `query` | string | `string` |  | ✓ |  | start time of interval |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#blobber-challenges-200) | OK | Challenges |  | [schema](#blobber-challenges-200-schema) |
| [400](#blobber-challenges-400) | Bad Request |  |  | [schema](#blobber-challenges-400-schema) |
| [404](#blobber-challenges-404) | Not Found |  |  | [schema](#blobber-challenges-404-schema) |
| [500](#blobber-challenges-500) | Internal Server Error |  |  | [schema](#blobber-challenges-500-schema) |

#### Responses


##### <span id="blobber-challenges-200"></span> 200 - Challenges
Status: OK

###### <span id="blobber-challenges-200-schema"></span> Schema
   
  


 [Challenges](#challenges)

##### <span id="blobber-challenges-400"></span> 400
Status: Bad Request

###### <span id="blobber-challenges-400-schema"></span> Schema

##### <span id="blobber-challenges-404"></span> 404
Status: Not Found

###### <span id="blobber-challenges-404-schema"></span> Schema

##### <span id="blobber-challenges-500"></span> 500
Status: Internal Server Error

###### <span id="blobber-challenges-500-schema"></span> Schema

### <span id="blobber-rank"></span> Gets the rank of a blobber. (*blobber-rank*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/blobber-rank
```

challenges passed / total challenges

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| id | `query` | string | `string` |  | ✓ |  | id of blobber |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#blobber-rank-200) | OK | Int64Map |  | [schema](#blobber-rank-200-schema) |
| [400](#blobber-rank-400) | Bad Request |  |  | [schema](#blobber-rank-400-schema) |

#### Responses


##### <span id="blobber-rank-200"></span> 200 - Int64Map
Status: OK

###### <span id="blobber-rank-200-schema"></span> Schema
   
  

[Int64Map](#int64-map)

##### <span id="blobber-rank-400"></span> 400
Status: Bad Request

###### <span id="blobber-rank-400-schema"></span> Schema

### <span id="blobber-ids"></span> blobber ids (*blobber_ids*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/blobber_ids
```

convert list of blobber urls into ids

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| free_allocation_data | `query` | string | `string` |  | ✓ |  | allocation data |
| limit | `query` | string | `string` |  |  |  | limit |
| offset | `query` | string | `string` |  |  |  | offset |
| sort | `query` | string | `string` |  |  |  | desc or asc |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#blobber-ids-200) | OK | stringArray |  | [schema](#blobber-ids-200-schema) |
| [400](#blobber-ids-400) | Bad Request |  |  | [schema](#blobber-ids-400-schema) |

#### Responses


##### <span id="blobber-ids-200"></span> 200 - stringArray
Status: OK

###### <span id="blobber-ids-200-schema"></span> Schema
   
  


 [StringArray](#string-array)

##### <span id="blobber-ids-400"></span> 400
Status: Bad Request

###### <span id="blobber-ids-400-schema"></span> Schema

### <span id="blobbers-by-geolocation"></span> Returns a list of all blobbers within a rectangle defined by maximum and minimum latitude and longitude values. (*blobbers-by-geolocation*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/blobbers-by-geolocation
```

+ name: max_latitude
description: maximum latitude value, defaults to 90
in: query
type: string
+ name: min_latitude
description:  minimum latitude value, defaults to -90
in: query
type: string
+ name: max_longitude
description: maximum max_longitude value, defaults to 180
in: query
type: string
+ name: min_longitude
description: minimum max_longitude value, defaults to -180
in: query
type: string
+ name: offset
description: offset
in: query
type: string
+ name: limit
description: limit
in: query
type: string
+ name: sort
description: desc or asc
in: query
type: string

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#blobbers-by-geolocation-200) | OK | stringArray |  | [schema](#blobbers-by-geolocation-200-schema) |
| [500](#blobbers-by-geolocation-500) | Internal Server Error |  |  | [schema](#blobbers-by-geolocation-500-schema) |

#### Responses


##### <span id="blobbers-by-geolocation-200"></span> 200 - stringArray
Status: OK

###### <span id="blobbers-by-geolocation-200-schema"></span> Schema
   
  


 [StringArray](#string-array)

##### <span id="blobbers-by-geolocation-500"></span> 500
Status: Internal Server Error

###### <span id="blobbers-by-geolocation-500-schema"></span> Schema

### <span id="blobbers-by-rank"></span> blobbers by rank (*blobbers-by-rank*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/blobbers-by-rank
```

Gets list of all blobbers ordered by rank

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| limit | `query` | string | `string` |  |  |  | limit |
| offset | `query` | string | `string` |  |  |  | offset |
| sort | `query` | string | `string` |  |  |  | desc or asc |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#blobbers-by-rank-200) | OK | storageNodeResponse |  | [schema](#blobbers-by-rank-200-schema) |
| [500](#blobbers-by-rank-500) | Internal Server Error |  |  | [schema](#blobbers-by-rank-500-schema) |

#### Responses


##### <span id="blobbers-by-rank-200"></span> 200 - storageNodeResponse
Status: OK

###### <span id="blobbers-by-rank-200-schema"></span> Schema
   
  

[StorageNodeResponse](#storage-node-response)

##### <span id="blobbers-by-rank-500"></span> 500
Status: Internal Server Error

###### <span id="blobbers-by-rank-500-schema"></span> Schema

### <span id="block"></span> block (*block*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/block
```

Gets block information

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| block_hash | `query` | string | `string` |  |  |  | block hash |
| date | `query` | string | `string` |  |  |  | block created closest to the date (epoch timestamp in nanoseconds) |
| round | `query` | string | `string` |  |  |  | block round |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#block-200) | OK | Block |  | [schema](#block-200-schema) |
| [400](#block-400) | Bad Request |  |  | [schema](#block-400-schema) |
| [500](#block-500) | Internal Server Error |  |  | [schema](#block-500-schema) |

#### Responses


##### <span id="block-200"></span> 200 - Block
Status: OK

###### <span id="block-200-schema"></span> Schema
   
  

[Block](#block)

##### <span id="block-400"></span> 400
Status: Bad Request

###### <span id="block-400-schema"></span> Schema

##### <span id="block-500"></span> 500
Status: Internal Server Error

###### <span id="block-500-schema"></span> Schema

### <span id="chainstatus"></span> chainstatus (*chainstatus*)

```
GET /v1/chain/get/stats
```

a handler to provide block statistics

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#chainstatus-200) | OK |  |  | [schema](#chainstatus-200-schema) |
| [404](#chainstatus-404) | Not Found |  |  | [schema](#chainstatus-404-schema) |

#### Responses


##### <span id="chainstatus-200"></span> 200
Status: OK

###### <span id="chainstatus-200-schema"></span> Schema

##### <span id="chainstatus-404"></span> 404
Status: Not Found

###### <span id="chainstatus-404-schema"></span> Schema

### <span id="collected-reward"></span> Returns collected reward for a client_id.
> Note: start-date and end-date resolves to the closest block number for those timestamps on the network. (*collected_reward*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/collected_reward
```

> Note: Using start/end-block and start/end-date together would only return results with start/end-block

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| client-id | `query` | string | `string` |  | ✓ |  | client id |
| data-points | `query` | string | `string` |  |  |  | number of data points in response |
| end-block | `query` | string | `string` |  |  |  | end block |
| end-date | `query` | string | `string` |  |  |  | end date |
| start-block | `query` | string | `string` |  |  |  | start block |
| start-date | `query` | string | `string` |  |  |  | start date |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#collected-reward-200) | OK | challengePoolStat |  | [schema](#collected-reward-200-schema) |
| [400](#collected-reward-400) | Bad Request |  |  | [schema](#collected-reward-400-schema) |

#### Responses


##### <span id="collected-reward-200"></span> 200 - challengePoolStat
Status: OK

###### <span id="collected-reward-200-schema"></span> Schema
   
  

[ChallengePoolStat](#challenge-pool-stat)

##### <span id="collected-reward-400"></span> 400
Status: Bad Request

###### <span id="collected-reward-400-schema"></span> Schema

### <span id="configs"></span> configs (*configs*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/configs
```

list minersc config settings

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#configs-200) | OK | StringMap |  | [schema](#configs-200-schema) |
| [400](#configs-400) | Bad Request |  |  | [schema](#configs-400-schema) |
| [484](#configs-484) | Status 484 |  |  | [schema](#configs-484-schema) |

#### Responses


##### <span id="configs-200"></span> 200 - StringMap
Status: OK

###### <span id="configs-200-schema"></span> Schema
   
  

[StringMap](#string-map)

##### <span id="configs-400"></span> 400
Status: Bad Request

###### <span id="configs-400-schema"></span> Schema

##### <span id="configs-484"></span> 484
Status: Status 484

###### <span id="configs-484-schema"></span> Schema

### <span id="count-readmarkers"></span> count readmarkers (*count_readmarkers*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/count_readmarkers
```

Gets read markers according to a filter

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| allocation_id | `query` | string | `string` |  | ✓ |  | count read markers for this allocation |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#count-readmarkers-200) | OK | readMarkersCount |  | [schema](#count-readmarkers-200-schema) |
| [500](#count-readmarkers-500) | Internal Server Error |  |  | [schema](#count-readmarkers-500-schema) |

#### Responses


##### <span id="count-readmarkers-200"></span> 200 - readMarkersCount
Status: OK

###### <span id="count-readmarkers-200-schema"></span> Schema
   
  

[ReadMarkersCount](#read-markers-count)

##### <span id="count-readmarkers-500"></span> 500
Status: Internal Server Error

###### <span id="count-readmarkers-500-schema"></span> Schema

### <span id="delegate-rewards"></span> delegate rewards (*delegate-rewards*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/delegate-rewards
```

Gets list of delegate rewards satisfying filter

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| end | `query` | string | `string` |  | ✓ |  | end time of interval |
| is_descending | `query` | string | `string` |  |  |  | is descending |
| limit | `query` | string | `string` |  |  |  | limit |
| offset | `query` | string | `string` |  |  |  | offset |
| start | `query` | string | `string` |  | ✓ |  | start time of interval |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#delegate-rewards-200) | OK | WriteMarker |  | [schema](#delegate-rewards-200-schema) |
| [400](#delegate-rewards-400) | Bad Request |  |  | [schema](#delegate-rewards-400-schema) |
| [500](#delegate-rewards-500) | Internal Server Error |  |  | [schema](#delegate-rewards-500-schema) |

#### Responses


##### <span id="delegate-rewards-200"></span> 200 - WriteMarker
Status: OK

###### <span id="delegate-rewards-200-schema"></span> Schema
   
  

[][WriteMarker](#write-marker)

##### <span id="delegate-rewards-400"></span> 400
Status: Bad Request

###### <span id="delegate-rewards-400-schema"></span> Schema

##### <span id="delegate-rewards-500"></span> 500
Status: Internal Server Error

###### <span id="delegate-rewards-500-schema"></span> Schema

### <span id="errors"></span> errors (*errors*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/errors
```

Gets errors returned by indicated transaction

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| limit | `query` | string | `string` |  |  |  | limit |
| offset | `query` | string | `string` |  |  |  | offset |
| sort | `query` | string | `string` |  |  |  | desc or asc |
| transaction_hash | `query` | string | `string` |  | ✓ |  | transaction_hash |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#errors-200) | OK | Error |  | [schema](#errors-200-schema) |
| [400](#errors-400) | Bad Request |  |  | [schema](#errors-400-schema) |
| [500](#errors-500) | Internal Server Error |  |  | [schema](#errors-500-schema) |

#### Responses


##### <span id="errors-200"></span> 200 - Error
Status: OK

###### <span id="errors-200-schema"></span> Schema
   
  

[][Error](#error)

##### <span id="errors-400"></span> 400
Status: Bad Request

###### <span id="errors-400-schema"></span> Schema

##### <span id="errors-500"></span> 500
Status: Internal Server Error

###### <span id="errors-500-schema"></span> Schema

### <span id="faucet-config"></span> faucet config (*faucet_config*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d3/faucet_config
```

faucet smart contract configuration settings

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#faucet-config-200) | OK | StringMap |  | [schema](#faucet-config-200-schema) |
| [404](#faucet-config-404) | Not Found |  |  | [schema](#faucet-config-404-schema) |

#### Responses


##### <span id="faucet-config-200"></span> 200 - StringMap
Status: OK

###### <span id="faucet-config-200-schema"></span> Schema
   
  

[StringMap](#string-map)

##### <span id="faucet-config-404"></span> 404
Status: Not Found

###### <span id="faucet-config-404-schema"></span> Schema

### <span id="free-alloc-blobbers"></span> returns list of all blobbers alive that match the free allocation request. (*free_alloc_blobbers*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/free_alloc_blobbers
```

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| free_allocation_data | `query` | string | `string` |  | ✓ |  | allocation data |
| limit | `query` | string | `string` |  |  |  | limit |
| offset | `query` | string | `string` |  |  |  | offset |
| sort | `query` | string | `string` |  |  |  | desc or asc |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#free-alloc-blobbers-200) | OK |  |  | [schema](#free-alloc-blobbers-200-schema) |
| [400](#free-alloc-blobbers-400) | Bad Request |  |  | [schema](#free-alloc-blobbers-400-schema) |

#### Responses


##### <span id="free-alloc-blobbers-200"></span> 200
Status: OK

###### <span id="free-alloc-blobbers-200-schema"></span> Schema

##### <span id="free-alloc-blobbers-400"></span> 400
Status: Bad Request

###### <span id="free-alloc-blobbers-400-schema"></span> Schema

### <span id="get-alloc-blobber-terms"></span> get alloc blobber terms (*getAllocBlobberTerms*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/alloc-blobber-term
```

Gets statistic for all locked tokens of a stake pool

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| allocation_id | `query` | string | `string` |  |  |  | id of allocation |
| blobber_id | `query` | string | `string` |  |  |  | id of blobber |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#get-alloc-blobber-terms-200) | OK | Terms |  | [schema](#get-alloc-blobber-terms-200-schema) |
| [400](#get-alloc-blobber-terms-400) | Bad Request |  |  | [schema](#get-alloc-blobber-terms-400-schema) |
| [500](#get-alloc-blobber-terms-500) | Internal Server Error |  |  | [schema](#get-alloc-blobber-terms-500-schema) |

#### Responses


##### <span id="get-alloc-blobber-terms-200"></span> 200 - Terms
Status: OK

###### <span id="get-alloc-blobber-terms-200-schema"></span> Schema
   
  

[Terms](#terms)

##### <span id="get-alloc-blobber-terms-400"></span> 400
Status: Bad Request

###### <span id="get-alloc-blobber-terms-400-schema"></span> Schema

##### <span id="get-alloc-blobber-terms-500"></span> 500
Status: Internal Server Error

###### <span id="get-alloc-blobber-terms-500-schema"></span> Schema

### <span id="get-authorizer"></span> get authorizer (*getAuthorizer*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d3/getAuthorizer
```

get authorizer

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#get-authorizer-200) | OK | authorizerResponse |  | [schema](#get-authorizer-200-schema) |
| [404](#get-authorizer-404) | Not Found |  |  | [schema](#get-authorizer-404-schema) |

#### Responses


##### <span id="get-authorizer-200"></span> 200 - authorizerResponse
Status: OK

###### <span id="get-authorizer-200-schema"></span> Schema
   
  

[AuthorizerResponse](#authorizer-response)

##### <span id="get-authorizer-404"></span> 404
Status: Not Found

###### <span id="get-authorizer-404-schema"></span> Schema

### <span id="get-authorizer-nodes"></span> get authorizer nodes (*getAuthorizerNodes*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712e0/getAuthorizerNodes
```

get authorizer nodes

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#get-authorizer-nodes-200) | OK | authorizerNodesResponse |  | [schema](#get-authorizer-nodes-200-schema) |
| [404](#get-authorizer-nodes-404) | Not Found |  |  | [schema](#get-authorizer-nodes-404-schema) |

#### Responses


##### <span id="get-authorizer-nodes-200"></span> 200 - authorizerNodesResponse
Status: OK

###### <span id="get-authorizer-nodes-200-schema"></span> Schema
   
  

[AuthorizerNodesResponse](#authorizer-nodes-response)

##### <span id="get-authorizer-nodes-404"></span> 404
Status: Not Found

###### <span id="get-authorizer-nodes-404-schema"></span> Schema

### <span id="get-blobber"></span> get blobber (*getBlobber*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/getBlobber
```

Get blobber information

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| blobber_id | `query` | string | `string` |  | ✓ |  | blobber for which to return information |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#get-blobber-200) | OK | storageNodeResponse |  | [schema](#get-blobber-200-schema) |
| [400](#get-blobber-400) | Bad Request |  |  | [schema](#get-blobber-400-schema) |
| [500](#get-blobber-500) | Internal Server Error |  |  | [schema](#get-blobber-500-schema) |

#### Responses


##### <span id="get-blobber-200"></span> 200 - storageNodeResponse
Status: OK

###### <span id="get-blobber-200-schema"></span> Schema
   
  

[StorageNodeResponse](#storage-node-response)

##### <span id="get-blobber-400"></span> 400
Status: Bad Request

###### <span id="get-blobber-400-schema"></span> Schema

##### <span id="get-blobber-500"></span> 500
Status: Internal Server Error

###### <span id="get-blobber-500-schema"></span> Schema

### <span id="get-challenge"></span> get challenge (*getChallenge*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/getchallenge
```

Gets challenges for a blobber by challenge id

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| blobber | `query` | string | `string` |  | ✓ |  | id of blobber |
| challenge | `query` | string | `string` |  | ✓ |  | id of challenge |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#get-challenge-200) | OK | StorageChallengeResponse |  | [schema](#get-challenge-200-schema) |
| [400](#get-challenge-400) | Bad Request |  |  | [schema](#get-challenge-400-schema) |
| [404](#get-challenge-404) | Not Found |  |  | [schema](#get-challenge-404-schema) |
| [500](#get-challenge-500) | Internal Server Error |  |  | [schema](#get-challenge-500-schema) |

#### Responses


##### <span id="get-challenge-200"></span> 200 - StorageChallengeResponse
Status: OK

###### <span id="get-challenge-200-schema"></span> Schema
   
  

[StorageChallengeResponse](#storage-challenge-response)

##### <span id="get-challenge-400"></span> 400
Status: Bad Request

###### <span id="get-challenge-400-schema"></span> Schema

##### <span id="get-challenge-404"></span> 404
Status: Not Found

###### <span id="get-challenge-404-schema"></span> Schema

##### <span id="get-challenge-500"></span> 500
Status: Internal Server Error

###### <span id="get-challenge-500-schema"></span> Schema

### <span id="get-challenge-pool-stat"></span> get challenge pool stat (*getChallengePoolStat*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/getChallengePoolStat
```

statistic for all locked tokens of a challenge pool

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| allocation_id | `query` | string | `string` |  | ✓ |  | allocation for which to get challenge pools statistics |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#get-challenge-pool-stat-200) | OK | challengePoolStat |  | [schema](#get-challenge-pool-stat-200-schema) |
| [400](#get-challenge-pool-stat-400) | Bad Request |  |  | [schema](#get-challenge-pool-stat-400-schema) |

#### Responses


##### <span id="get-challenge-pool-stat-200"></span> 200 - challengePoolStat
Status: OK

###### <span id="get-challenge-pool-stat-200-schema"></span> Schema
   
  

[ChallengePoolStat](#challenge-pool-stat)

##### <span id="get-challenge-pool-stat-400"></span> 400
Status: Bad Request

###### <span id="get-challenge-pool-stat-400-schema"></span> Schema

### <span id="get-client-pools"></span> get client pools (*getClientPools*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/getClientPools
```

get client pools

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#get-client-pools-200) | OK | vestingClientPools |  | [schema](#get-client-pools-200-schema) |
| [500](#get-client-pools-500) | Internal Server Error |  |  | [schema](#get-client-pools-500-schema) |

#### Responses


##### <span id="get-client-pools-200"></span> 200 - vestingClientPools
Status: OK

###### <span id="get-client-pools-200-schema"></span> Schema
   
  

[ClientPools](#client-pools)

##### <span id="get-client-pools-500"></span> 500
Status: Internal Server Error

###### <span id="get-client-pools-500-schema"></span> Schema

### <span id="get-dkg-list"></span> get dkg list (*getDkgList*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/getDkgList
```

gets dkg miners list

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

### <span id="get-events"></span> get events (*getEvents*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/getEvents
```

events for block

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| block_number | `query` | string | `string` |  |  |  | block number |
| limit | `query` | string | `string` |  |  |  | limit |
| offset | `query` | string | `string` |  |  |  | offset |
| sort | `query` | string | `string` |  |  |  | desc or asc |
| tag | `query` | string | `string` |  |  |  | tag |
| tx_hash | `query` | string | `string` |  |  |  | hash of transaction |
| type | `query` | string | `string` |  |  |  | type |

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

### <span id="get-group-share-or-signs"></span> get group share or signs (*getGroupShareOrSigns*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/getGroupShareOrSigns
```

gets group share or signs

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

### <span id="get-m-s-stake-pool-stat"></span> get m s stake pool stat (*getMSStakePoolStat*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/getStakePoolStat
```

Gets statistic for all locked tokens of a stake pool

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| provider_id | `query` | string | `string` |  | ✓ |  | id of a provider |
| provider_type | `query` | string | `string` |  | ✓ |  | type of the provider, ie: miner. sharder |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#get-m-s-stake-pool-stat-200) | OK | stakePoolStat |  | [schema](#get-m-s-stake-pool-stat-200-schema) |
| [400](#get-m-s-stake-pool-stat-400) | Bad Request |  |  | [schema](#get-m-s-stake-pool-stat-400-schema) |
| [500](#get-m-s-stake-pool-stat-500) | Internal Server Error |  |  | [schema](#get-m-s-stake-pool-stat-500-schema) |

#### Responses


##### <span id="get-m-s-stake-pool-stat-200"></span> 200 - stakePoolStat
Status: OK

###### <span id="get-m-s-stake-pool-stat-200-schema"></span> Schema
   
  

[StakePoolStat](#stake-pool-stat)

##### <span id="get-m-s-stake-pool-stat-400"></span> 400
Status: Bad Request

###### <span id="get-m-s-stake-pool-stat-400-schema"></span> Schema

##### <span id="get-m-s-stake-pool-stat-500"></span> 500
Status: Internal Server Error

###### <span id="get-m-s-stake-pool-stat-500-schema"></span> Schema

### <span id="get-magic-block"></span> get magic block (*getMagicBlock*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/getMagicBlock
```

gets magic block

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

### <span id="get-miner-list"></span> get miner list (*getMinerList*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/getMinerList
```

lists miners

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| active | `query` | string | `string` |  |  |  | active |
| limit | `query` | string | `string` |  |  |  | limit |
| offset | `query` | string | `string` |  |  |  | offset |
| sort | `query` | string | `string` |  |  |  | desc or asc |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#get-miner-list-200) | OK | InterfaceMap |  | [schema](#get-miner-list-200-schema) |
| [400](#get-miner-list-400) | Bad Request |  |  | [schema](#get-miner-list-400-schema) |
| [484](#get-miner-list-484) | Status 484 |  |  | [schema](#get-miner-list-484-schema) |

#### Responses


##### <span id="get-miner-list-200"></span> 200 - InterfaceMap
Status: OK

###### <span id="get-miner-list-200-schema"></span> Schema
   
  

[InterfaceMap](#interface-map)

##### <span id="get-miner-list-400"></span> 400
Status: Bad Request

###### <span id="get-miner-list-400-schema"></span> Schema

##### <span id="get-miner-list-484"></span> 484
Status: Status 484

###### <span id="get-miner-list-484-schema"></span> Schema

### <span id="get-mpks-list"></span> get mpks list (*getMpksList*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/getMpksList
```

gets dkg miners list

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

### <span id="get-nodepool"></span> get nodepool (*getNodepool*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/getNodepool
```

provides nodepool information for registered miners

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#get-nodepool-200) | OK | PoolMembersInfo |  | [schema](#get-nodepool-200-schema) |
| [400](#get-nodepool-400) | Bad Request |  |  | [schema](#get-nodepool-400-schema) |
| [484](#get-nodepool-484) | Status 484 |  |  | [schema](#get-nodepool-484-schema) |

#### Responses


##### <span id="get-nodepool-200"></span> 200 - PoolMembersInfo
Status: OK

###### <span id="get-nodepool-200-schema"></span> Schema
   
  

[PoolMembersInfo](#pool-members-info)

##### <span id="get-nodepool-400"></span> 400
Status: Bad Request

###### <span id="get-nodepool-400-schema"></span> Schema

##### <span id="get-nodepool-484"></span> 484
Status: Status 484

###### <span id="get-nodepool-484-schema"></span> Schema

### <span id="get-phase"></span> get phase (*getPhase*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/getPhase
```

get phase nodes

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

### <span id="get-pool-info"></span> get pool info (*getPoolInfo*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/getPoolInfo
```

get vesting configuration settings

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#get-pool-info-200) | OK | vestingInfo |  | [schema](#get-pool-info-200-schema) |
| [500](#get-pool-info-500) | Internal Server Error |  |  | [schema](#get-pool-info-500-schema) |

#### Responses


##### <span id="get-pool-info-200"></span> 200 - vestingInfo
Status: OK

###### <span id="get-pool-info-200-schema"></span> Schema
   
  

[Info](#info)

##### <span id="get-pool-info-500"></span> 500
Status: Internal Server Error

###### <span id="get-pool-info-500-schema"></span> Schema

### <span id="get-read-pool-stat"></span> get read pool stat (*getReadPoolStat*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/getReadPoolStat
```

Gets  statistic for all locked tokens of the read pool

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| client_id | `query` | string | `string` |  | ✓ |  | client for which to get read pools statistics |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#get-read-pool-stat-200) | OK | readPool |  | [schema](#get-read-pool-stat-200-schema) |
| [400](#get-read-pool-stat-400) | Bad Request |  |  | [schema](#get-read-pool-stat-400-schema) |

#### Responses


##### <span id="get-read-pool-stat-200"></span> 200 - readPool
Status: OK

###### <span id="get-read-pool-stat-200-schema"></span> Schema
   
  

[ReadPool](#read-pool)

##### <span id="get-read-pool-stat-400"></span> 400
Status: Bad Request

###### <span id="get-read-pool-stat-400-schema"></span> Schema

### <span id="get-sharder-keep-list"></span> get sharder keep list (*getSharderKeepList*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/getSharderKeepList
```

get total sharder stake

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

### <span id="get-sharder-list"></span> get sharder list (*getSharderList*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/getSharderList
```

lists sharders

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| active | `query` | string | `string` |  |  |  | active |
| limit | `query` | string | `string` |  |  |  | limit |
| offset | `query` | string | `string` |  |  |  | offset |
| sort | `query` | string | `string` |  |  |  | desc or asc |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#get-sharder-list-200) | OK | InterfaceMap |  | [schema](#get-sharder-list-200-schema) |
| [400](#get-sharder-list-400) | Bad Request |  |  | [schema](#get-sharder-list-400-schema) |
| [484](#get-sharder-list-484) | Status 484 |  |  | [schema](#get-sharder-list-484-schema) |

#### Responses


##### <span id="get-sharder-list-200"></span> 200 - InterfaceMap
Status: OK

###### <span id="get-sharder-list-200-schema"></span> Schema
   
  

[InterfaceMap](#interface-map)

##### <span id="get-sharder-list-400"></span> 400
Status: Bad Request

###### <span id="get-sharder-list-400-schema"></span> Schema

##### <span id="get-sharder-list-484"></span> 484
Status: Status 484

###### <span id="get-sharder-list-484-schema"></span> Schema

### <span id="get-stake-pool-stat"></span> get stake pool stat (*getStakePoolStat*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/getStakePoolStat
```

Gets statistic for all locked tokens of a stake pool

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| provider_id | `query` | string | `string` |  | ✓ |  | id of a provider |
| provider_type | `query` | string | `string` |  | ✓ |  | type of the provider, ie: blobber. validator |

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

### <span id="get-user-locked-total"></span> get user locked total (*getUserLockedTotal*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/getUserLockedTotal
```

Gets statistic for a user's stake pools

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| client_id | `query` | string | `string` |  | ✓ |  | client for which to get stake pool information |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#get-user-locked-total-200) | OK | userLockedTotalResponse |  | [schema](#get-user-locked-total-200-schema) |
| [400](#get-user-locked-total-400) | Bad Request |  |  | [schema](#get-user-locked-total-400-schema) |

#### Responses


##### <span id="get-user-locked-total-200"></span> 200 - userLockedTotalResponse
Status: OK

###### <span id="get-user-locked-total-200-schema"></span> Schema
   
  

[UserLockedTotalResponse](#user-locked-total-response)

##### <span id="get-user-locked-total-400"></span> 400
Status: Bad Request

###### <span id="get-user-locked-total-400-schema"></span> Schema

### <span id="get-user-pools"></span> get user pools (*getUserPools*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/getUserPools
```

user oriented pools requests handler

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| client_id | `query` | string | `string` |  | ✓ |  | client for which to get write pools statistics |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#get-user-pools-200) | OK | userPoolStat |  | [schema](#get-user-pools-200-schema) |
| [400](#get-user-pools-400) | Bad Request |  |  | [schema](#get-user-pools-400-schema) |
| [484](#get-user-pools-484) | Status 484 |  |  | [schema](#get-user-pools-484-schema) |

#### Responses


##### <span id="get-user-pools-200"></span> 200 - userPoolStat
Status: OK

###### <span id="get-user-pools-200-schema"></span> Schema
   
  

[UserPoolStat](#user-pool-stat)

##### <span id="get-user-pools-400"></span> 400
Status: Bad Request

###### <span id="get-user-pools-400-schema"></span> Schema

##### <span id="get-user-pools-484"></span> 484
Status: Status 484

###### <span id="get-user-pools-484-schema"></span> Schema

### <span id="get-user-stake-pool-stat"></span> get user stake pool stat (*getUserStakePoolStat*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/getUserStakePoolStat
```

Gets statistic for a user's stake pools

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| client_id | `query` | string | `string` |  | ✓ |  | client for which to get stake pool information |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#get-user-stake-pool-stat-200) | OK | userPoolStat |  | [schema](#get-user-stake-pool-stat-200-schema) |
| [400](#get-user-stake-pool-stat-400) | Bad Request |  |  | [schema](#get-user-stake-pool-stat-400-schema) |

#### Responses


##### <span id="get-user-stake-pool-stat-200"></span> 200 - userPoolStat
Status: OK

###### <span id="get-user-stake-pool-stat-200-schema"></span> Schema
   
  

[UserPoolStat](#user-pool-stat)

##### <span id="get-user-stake-pool-stat-400"></span> 400
Status: Bad Request

###### <span id="get-user-stake-pool-stat-400-schema"></span> Schema

### <span id="get-write-markers"></span> get write markers (*getWriteMarkers*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/getWriteMarkers
```

Gets read markers according to a filter

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| allocation_id | `query` | string | `string` |  | ✓ |  | count write markers for this allocation |
| filename | `query` | string | `string` |  | ✓ |  | file name |
| limit | `query` | string | `string` |  |  |  | limit |
| offset | `query` | string | `string` |  |  |  | offset |
| sort | `query` | string | `string` |  |  |  | desc or asc |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#get-write-markers-200) | OK | WriteMarker |  | [schema](#get-write-markers-200-schema) |
| [400](#get-write-markers-400) | Bad Request |  |  | [schema](#get-write-markers-400-schema) |
| [500](#get-write-markers-500) | Internal Server Error |  |  | [schema](#get-write-markers-500-schema) |

#### Responses


##### <span id="get-write-markers-200"></span> 200 - WriteMarker
Status: OK

###### <span id="get-write-markers-200-schema"></span> Schema
   
  

[][WriteMarker](#write-marker)

##### <span id="get-write-markers-400"></span> 400
Status: Bad Request

###### <span id="get-write-markers-400-schema"></span> Schema

##### <span id="get-write-markers-500"></span> 500
Status: Internal Server Error

###### <span id="get-write-markers-500-schema"></span> Schema

### <span id="get-blobber-total-stakes"></span> get blobber total stakes (*get_blobber_total_stakes*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/get_blobber_total_stakes
```

Gets total stake of all blobbers combined

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#get-blobber-total-stakes-200) | OK | Int64Map |  | [schema](#get-blobber-total-stakes-200-schema) |
| [500](#get-blobber-total-stakes-500) | Internal Server Error |  |  | [schema](#get-blobber-total-stakes-500-schema) |

#### Responses


##### <span id="get-blobber-total-stakes-200"></span> 200 - Int64Map
Status: OK

###### <span id="get-blobber-total-stakes-200-schema"></span> Schema
   
  

[Int64Map](#int64-map)

##### <span id="get-blobber-total-stakes-500"></span> 500
Status: Internal Server Error

###### <span id="get-blobber-total-stakes-500-schema"></span> Schema

### <span id="get-blocks"></span> Gets block information for all blocks. Todo: We need to add a filter to this. (*get_blocks*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/get_blocks
```

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| block_hash | `query` | string | `string` |  | ✓ |  | block hash |
| limit | `query` | string | `string` |  |  |  | limit |
| offset | `query` | string | `string` |  |  |  | offset |
| sort | `query` | string | `string` |  |  |  | desc or asc |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#get-blocks-200) | OK | fullBlock |  | [schema](#get-blocks-200-schema) |
| [400](#get-blocks-400) | Bad Request |  |  | [schema](#get-blocks-400-schema) |
| [500](#get-blocks-500) | Internal Server Error |  |  | [schema](#get-blocks-500-schema) |

#### Responses


##### <span id="get-blocks-200"></span> 200 - fullBlock
Status: OK

###### <span id="get-blocks-200-schema"></span> Schema
   
  

[][FullBlock](#full-block)

##### <span id="get-blocks-400"></span> 400
Status: Bad Request

###### <span id="get-blocks-400-schema"></span> Schema

##### <span id="get-blocks-500"></span> 500
Status: Internal Server Error

###### <span id="get-blocks-500-schema"></span> Schema

### <span id="get-miner-geolocations"></span> get miner geolocations (*get_miner_geolocations*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/get_miner_geolocations
```

list minersc config settings

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| active | `query` | string | `string` |  | ✓ |  | active |
| limit | `query` | string | `string` |  | ✓ |  | limit |
| offset | `query` | string | `string` |  | ✓ |  | offset |
| sort | `query` | string | `string` |  |  |  | desc or asc |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#get-miner-geolocations-200) | OK | MinerGeolocation |  | [schema](#get-miner-geolocations-200-schema) |
| [400](#get-miner-geolocations-400) | Bad Request |  |  | [schema](#get-miner-geolocations-400-schema) |
| [484](#get-miner-geolocations-484) | Status 484 |  |  | [schema](#get-miner-geolocations-484-schema) |

#### Responses


##### <span id="get-miner-geolocations-200"></span> 200 - MinerGeolocation
Status: OK

###### <span id="get-miner-geolocations-200-schema"></span> Schema
   
  

[MinerGeolocation](#miner-geolocation)

##### <span id="get-miner-geolocations-400"></span> 400
Status: Bad Request

###### <span id="get-miner-geolocations-400-schema"></span> Schema

##### <span id="get-miner-geolocations-484"></span> 484
Status: Status 484

###### <span id="get-miner-geolocations-484-schema"></span> Schema

### <span id="get-miners-stake"></span> get miners stake (*get_miners_stake*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/get_miners_stake
```

get total miner stake

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#get-miners-stake-200) | OK | Int64Map |  | [schema](#get-miners-stake-200-schema) |
| [404](#get-miners-stake-404) | Not Found |  |  | [schema](#get-miners-stake-404-schema) |

#### Responses


##### <span id="get-miners-stake-200"></span> 200 - Int64Map
Status: OK

###### <span id="get-miners-stake-200-schema"></span> Schema
   
  

[Int64Map](#int64-map)

##### <span id="get-miners-stake-404"></span> 404
Status: Not Found

###### <span id="get-miners-stake-404-schema"></span> Schema

### <span id="get-miners-stats"></span> get miners stats (*get_miners_stats*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/get_miners_stats
```

get count of active and inactive miners

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

### <span id="get-sharder-geolocations"></span> get sharder geolocations (*get_sharder_geolocations*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/get_sharder_geolocations
```

list minersc config settings

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| active | `query` | string | `string` |  | ✓ |  | active |
| limit | `query` | string | `string` |  | ✓ |  | limit |
| offset | `query` | string | `string` |  | ✓ |  | offset |
| sort | `query` | string | `string` |  |  |  | desc or asc |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#get-sharder-geolocations-200) | OK | SharderGeolocation |  | [schema](#get-sharder-geolocations-200-schema) |
| [400](#get-sharder-geolocations-400) | Bad Request |  |  | [schema](#get-sharder-geolocations-400-schema) |
| [484](#get-sharder-geolocations-484) | Status 484 |  |  | [schema](#get-sharder-geolocations-484-schema) |

#### Responses


##### <span id="get-sharder-geolocations-200"></span> 200 - SharderGeolocation
Status: OK

###### <span id="get-sharder-geolocations-200-schema"></span> Schema
   
  

[SharderGeolocation](#sharder-geolocation)

##### <span id="get-sharder-geolocations-400"></span> 400
Status: Bad Request

###### <span id="get-sharder-geolocations-400-schema"></span> Schema

##### <span id="get-sharder-geolocations-484"></span> 484
Status: Status 484

###### <span id="get-sharder-geolocations-484-schema"></span> Schema

### <span id="get-sharders-stake"></span> get sharders stake (*get_sharders_stake*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/get_sharders_stake
```

get total sharder stake

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#get-sharders-stake-200) | OK | Int64Map |  | [schema](#get-sharders-stake-200-schema) |
| [404](#get-sharders-stake-404) | Not Found |  |  | [schema](#get-sharders-stake-404-schema) |

#### Responses


##### <span id="get-sharders-stake-200"></span> 200 - Int64Map
Status: OK

###### <span id="get-sharders-stake-200-schema"></span> Schema
   
  

[Int64Map](#int64-map)

##### <span id="get-sharders-stake-404"></span> 404
Status: Not Found

###### <span id="get-sharders-stake-404-schema"></span> Schema

### <span id="get-sharders-stats"></span> get sharders stats (*get_sharders_stats*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/get_sharders_stats
```

get count of active and inactive miners

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

### <span id="get-validator"></span> get validator (*get_validator*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/get_validator
```

Gets validator information

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| validator_id | `query` | string | `string` |  | ✓ |  | validator on which to get information |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#get-validator-200) | OK | Validator |  | [schema](#get-validator-200-schema) |
| [400](#get-validator-400) | Bad Request |  |  | [schema](#get-validator-400-schema) |
| [500](#get-validator-500) | Internal Server Error |  |  | [schema](#get-validator-500-schema) |

#### Responses


##### <span id="get-validator-200"></span> 200 - Validator
Status: OK

###### <span id="get-validator-200-schema"></span> Schema
   
  

[Validator](#validator)

##### <span id="get-validator-400"></span> 400
Status: Bad Request

###### <span id="get-validator-400-schema"></span> Schema

##### <span id="get-validator-500"></span> 500
Status: Internal Server Error

###### <span id="get-validator-500-schema"></span> Schema

### <span id="getblobbers"></span> Gets list of all blobbers alive (e.g. excluding blobbers with zero capacity). (*getblobbers*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/getblobbers
```

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| limit | `query` | string | `string` |  |  |  | limit |
| offset | `query` | string | `string` |  |  |  | offset |
| sort | `query` | string | `string` |  |  |  | desc or asc |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#getblobbers-200) | OK | storageNodesResponse |  | [schema](#getblobbers-200-schema) |
| [500](#getblobbers-500) | Internal Server Error |  |  | [schema](#getblobbers-500-schema) |

#### Responses


##### <span id="getblobbers-200"></span> 200 - storageNodesResponse
Status: OK

###### <span id="getblobbers-200-schema"></span> Schema
   
  

[StorageNodesResponse](#storage-nodes-response)

##### <span id="getblobbers-500"></span> 500
Status: Internal Server Error

###### <span id="getblobbers-500-schema"></span> Schema

### <span id="global-periodic-limit"></span> global periodic limit (*globalPeriodicLimit*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d3/globalPeriodicLimit
```

list minersc config settings

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#global-periodic-limit-200) | OK | periodicResponse |  | [schema](#global-periodic-limit-200-schema) |
| [404](#global-periodic-limit-404) | Not Found |  |  | [schema](#global-periodic-limit-404-schema) |

#### Responses


##### <span id="global-periodic-limit-200"></span> 200 - periodicResponse
Status: OK

###### <span id="global-periodic-limit-200-schema"></span> Schema
   
  

[PeriodicResponse](#periodic-response)

##### <span id="global-periodic-limit-404"></span> 404
Status: Not Found

###### <span id="global-periodic-limit-404-schema"></span> Schema

### <span id="global-settings"></span> global settings (*globalSettings*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/globalSettings
```

global object for miner smart contracts

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#global-settings-200) | OK | MinerGlobalSettings |  | [schema](#global-settings-200-schema) |
| [400](#global-settings-400) | Bad Request |  |  | [schema](#global-settings-400-schema) |

#### Responses


##### <span id="global-settings-200"></span> 200 - MinerGlobalSettings
Status: OK

###### <span id="global-settings-200-schema"></span> Schema
   
  

[GlobalSettings](#global-settings)

##### <span id="global-settings-400"></span> 400
Status: Bad Request

###### <span id="global-settings-400-schema"></span> Schema

### <span id="latestreadmarker"></span> latestreadmarker (*latestreadmarker*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/latestreadmarker
```

Gets latest read marker for a client and blobber

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| blobber | `query` | string | `string` |  |  |  | blobber |
| client | `query` | string | `string` |  |  |  | client |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#latestreadmarker-200) | OK | ReadMarker |  | [schema](#latestreadmarker-200-schema) |
| [500](#latestreadmarker-500) | Internal Server Error |  |  | [schema](#latestreadmarker-500-schema) |

#### Responses


##### <span id="latestreadmarker-200"></span> 200 - ReadMarker
Status: OK

###### <span id="latestreadmarker-200-schema"></span> Schema
   
  

[ReadMarker](#read-marker)

##### <span id="latestreadmarker-500"></span> 500
Status: Internal Server Error

###### <span id="latestreadmarker-500-schema"></span> Schema

### <span id="node-pool-stat"></span> node pool stat (*nodePoolStat*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/nodePoolStat
```

lists sharders

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| id | `query` | string | `string` |  | ✓ |  | id |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#node-pool-stat-200) | OK |  |  | [schema](#node-pool-stat-200-schema) |
| [400](#node-pool-stat-400) | Bad Request |  |  | [schema](#node-pool-stat-400-schema) |
| [484](#node-pool-stat-484) | Status 484 |  |  | [schema](#node-pool-stat-484-schema) |

#### Responses


##### <span id="node-pool-stat-200"></span> 200
Status: OK

###### <span id="node-pool-stat-200-schema"></span> Schema

##### <span id="node-pool-stat-400"></span> 400
Status: Bad Request

###### <span id="node-pool-stat-400-schema"></span> Schema

##### <span id="node-pool-stat-484"></span> 484
Status: Status 484

###### <span id="node-pool-stat-484-schema"></span> Schema

### <span id="node-stat"></span> node stat (*nodeStat*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/nodeStat
```

lists sharders

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| id | `query` | string | `string` |  | ✓ |  | id |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#node-stat-200) | OK | nodeStat |  | [schema](#node-stat-200-schema) |
| [400](#node-stat-400) | Bad Request |  |  | [schema](#node-stat-400-schema) |
| [484](#node-stat-484) | Status 484 |  |  | [schema](#node-stat-484-schema) |

#### Responses


##### <span id="node-stat-200"></span> 200 - nodeStat
Status: OK

###### <span id="node-stat-200-schema"></span> Schema
   
  

[NodeStat](#node-stat)

##### <span id="node-stat-400"></span> 400
Status: Bad Request

###### <span id="node-stat-400-schema"></span> Schema

##### <span id="node-stat-484"></span> 484
Status: Status 484

###### <span id="node-stat-484-schema"></span> Schema

### <span id="openchallenges"></span> openchallenges (*openchallenges*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/openchallenges
```

Gets open challenges for a blobber

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| blobber | `query` | string | `string` |  | ✓ |  | id of blobber for which to get open challenges |
| limit | `query` | string | `string` |  |  |  | limit |
| offset | `query` | string | `string` |  |  |  | offset |
| sort | `query` | string | `string` |  |  |  | desc or asc |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#openchallenges-200) | OK | ChallengesResponse |  | [schema](#openchallenges-200-schema) |
| [400](#openchallenges-400) | Bad Request |  |  | [schema](#openchallenges-400-schema) |
| [404](#openchallenges-404) | Not Found |  |  | [schema](#openchallenges-404-schema) |
| [500](#openchallenges-500) | Internal Server Error |  |  | [schema](#openchallenges-500-schema) |

#### Responses


##### <span id="openchallenges-200"></span> 200 - ChallengesResponse
Status: OK

###### <span id="openchallenges-200-schema"></span> Schema
   
  

[ChallengesResponse](#challenges-response)

##### <span id="openchallenges-400"></span> 400
Status: Bad Request

###### <span id="openchallenges-400-schema"></span> Schema

##### <span id="openchallenges-404"></span> 404
Status: Not Found

###### <span id="openchallenges-404-schema"></span> Schema

##### <span id="openchallenges-500"></span> 500
Status: Internal Server Error

###### <span id="openchallenges-500-schema"></span> Schema

### <span id="personal-periodic-limit"></span> personal periodic limit (*personalPeriodicLimit*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d3/personalPeriodicLimit
```

list minersc config settings

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#personal-periodic-limit-200) | OK | periodicResponse |  | [schema](#personal-periodic-limit-200-schema) |
| [404](#personal-periodic-limit-404) | Not Found |  |  | [schema](#personal-periodic-limit-404-schema) |

#### Responses


##### <span id="personal-periodic-limit-200"></span> 200 - periodicResponse
Status: OK

###### <span id="personal-periodic-limit-200-schema"></span> Schema
   
  

[PeriodicResponse](#periodic-response)

##### <span id="personal-periodic-limit-404"></span> 404
Status: Not Found

###### <span id="personal-periodic-limit-404-schema"></span> Schema

### <span id="pour-amount"></span> pour amount (*pourAmount*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d3/pourAmount
```

pour amount

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#pour-amount-200) | OK |  |  | [schema](#pour-amount-200-schema) |
| [404](#pour-amount-404) | Not Found |  |  | [schema](#pour-amount-404-schema) |

#### Responses


##### <span id="pour-amount-200"></span> 200
Status: OK

###### <span id="pour-amount-200-schema"></span> Schema

##### <span id="pour-amount-404"></span> 404
Status: Not Found

###### <span id="pour-amount-404-schema"></span> Schema

### <span id="provider-rewards"></span> provider rewards (*provider-rewards*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/provider-rewards
```

Gets list of provider rewards satisfying filter

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| end | `query` | string | `string` |  | ✓ |  | end time of interval |
| is_descending | `query` | string | `string` |  |  |  | is descending |
| limit | `query` | string | `string` |  |  |  | limit |
| offset | `query` | string | `string` |  |  |  | offset |
| start | `query` | string | `string` |  | ✓ |  | start time of interval |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#provider-rewards-200) | OK | WriteMarker |  | [schema](#provider-rewards-200-schema) |
| [400](#provider-rewards-400) | Bad Request |  |  | [schema](#provider-rewards-400-schema) |
| [500](#provider-rewards-500) | Internal Server Error |  |  | [schema](#provider-rewards-500-schema) |

#### Responses


##### <span id="provider-rewards-200"></span> 200 - WriteMarker
Status: OK

###### <span id="provider-rewards-200-schema"></span> Schema
   
  

[][WriteMarker](#write-marker)

##### <span id="provider-rewards-400"></span> 400
Status: Bad Request

###### <span id="provider-rewards-400-schema"></span> Schema

##### <span id="provider-rewards-500"></span> 500
Status: Internal Server Error

###### <span id="provider-rewards-500-schema"></span> Schema

### <span id="readmarkers"></span> readmarkers (*readmarkers*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/readmarkers
```

Gets read markers according to a filter

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| allocation_id | `query` | string | `string` |  |  |  | filter read markers by this allocation |
| auth_ticket | `query` | string | `string` |  |  |  | filter in only read markers using auth thicket |
| limit | `query` | string | `string` |  |  |  | limit |
| offset | `query` | string | `string` |  |  |  | offset |
| sort | `query` | string | `string` |  |  |  | desc or asc |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#readmarkers-200) | OK | ReadMarker |  | [schema](#readmarkers-200-schema) |
| [500](#readmarkers-500) | Internal Server Error |  |  | [schema](#readmarkers-500-schema) |

#### Responses


##### <span id="readmarkers-200"></span> 200 - ReadMarker
Status: OK

###### <span id="readmarkers-200-schema"></span> Schema
   
  

[][ReadMarker](#read-marker)

##### <span id="readmarkers-500"></span> 500
Status: Internal Server Error

###### <span id="readmarkers-500-schema"></span> Schema

### <span id="replicate-blobber-aggregates"></span> replicate blobber aggregates (*replicateBlobberAggregates*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/replicate-blobber-aggregate
```

Gets list of blobber aggregate records

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| limit | `query` | string | `string` |  |  |  | limit |
| offset | `query` | string | `string` |  |  |  | offset |
| sort | `query` | string | `string` |  |  |  | desc or asc |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#replicate-blobber-aggregates-200) | OK | StringMap |  | [schema](#replicate-blobber-aggregates-200-schema) |
| [500](#replicate-blobber-aggregates-500) | Internal Server Error |  |  | [schema](#replicate-blobber-aggregates-500-schema) |

#### Responses


##### <span id="replicate-blobber-aggregates-200"></span> 200 - StringMap
Status: OK

###### <span id="replicate-blobber-aggregates-200-schema"></span> Schema
   
  

[StringMap](#string-map)

##### <span id="replicate-blobber-aggregates-500"></span> 500
Status: Internal Server Error

###### <span id="replicate-blobber-aggregates-500-schema"></span> Schema

### <span id="replicate-snapshots"></span> replicate snapshots (*replicateSnapshots*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/replicate-snapshots
```

Gets list of snapshot records

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| limit | `query` | string | `string` |  |  |  | limit |
| offset | `query` | string | `string` |  |  |  | offset |
| sort | `query` | string | `string` |  |  |  | desc or asc |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#replicate-snapshots-200) | OK | StringMap |  | [schema](#replicate-snapshots-200-schema) |
| [500](#replicate-snapshots-500) | Internal Server Error |  |  | [schema](#replicate-snapshots-500-schema) |

#### Responses


##### <span id="replicate-snapshots-200"></span> 200 - StringMap
Status: OK

###### <span id="replicate-snapshots-200-schema"></span> Schema
   
  

[StringMap](#string-map)

##### <span id="replicate-snapshots-500"></span> 500
Status: Internal Server Error

###### <span id="replicate-snapshots-500-schema"></span> Schema

### <span id="search"></span> search (*search*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/search
```

Generic search endpoint

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| searchString | `query` | string | `string` |  | ✓ |  | Generic query string, supported inputs: Block hash, Round num, Transaction hash, File name, Content hash, Wallet address |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#search-200) | OK |  |  | [schema](#search-200-schema) |
| [400](#search-400) | Bad Request |  |  | [schema](#search-400-schema) |
| [500](#search-500) | Internal Server Error |  |  | [schema](#search-500-schema) |

#### Responses


##### <span id="search-200"></span> 200
Status: OK

###### <span id="search-200-schema"></span> Schema

##### <span id="search-400"></span> 400
Status: Bad Request

###### <span id="search-400-schema"></span> Schema

##### <span id="search-500"></span> 500
Status: Internal Server Error

###### <span id="search-500-schema"></span> Schema

### <span id="sharderstats"></span> sharderstats (*sharderstats*)

```
GET /v1/sharder/get/stats
```

a handler to get sharder stats

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#sharderstats-200) | OK |  |  | [schema](#sharderstats-200-schema) |
| [404](#sharderstats-404) | Not Found |  |  | [schema](#sharderstats-404-schema) |

#### Responses


##### <span id="sharderstats-200"></span> 200
Status: OK

###### <span id="sharderstats-200-schema"></span> Schema

##### <span id="sharderstats-404"></span> 404
Status: Not Found

###### <span id="sharderstats-404-schema"></span> Schema

### <span id="storage-config"></span> storage config (*storage-config*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/storage-config
```

Gets the current storage smart contract settings

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#storage-config-200) | OK | StringMap |  | [schema](#storage-config-200-schema) |
| [400](#storage-config-400) | Bad Request |  |  | [schema](#storage-config-400-schema) |

#### Responses


##### <span id="storage-config-200"></span> 200 - StringMap
Status: OK

###### <span id="storage-config-200-schema"></span> Schema
   
  

[StringMap](#string-map)

##### <span id="storage-config-400"></span> 400
Status: Bad Request

###### <span id="storage-config-400-schema"></span> Schema

### <span id="timestamps-to-rounds"></span> timestamps to rounds (*timestampsToRounds*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/timestamp-to-round
```

Get round(s) number for timestamp(s)

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| timestamps | `query` | string | `string` |  | ✓ |  | timestamps you want to convert to rounds |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#timestamps-to-rounds-200) | OK | timestampToRoundResp |  | [schema](#timestamps-to-rounds-200-schema) |
| [400](#timestamps-to-rounds-400) | Bad Request |  |  | [schema](#timestamps-to-rounds-400-schema) |
| [500](#timestamps-to-rounds-500) | Internal Server Error |  |  | [schema](#timestamps-to-rounds-500-schema) |

#### Responses


##### <span id="timestamps-to-rounds-200"></span> 200 - timestampToRoundResp
Status: OK

###### <span id="timestamps-to-rounds-200-schema"></span> Schema
   
  

[TimestampToRoundResp](#timestamp-to-round-resp)

##### <span id="timestamps-to-rounds-400"></span> 400
Status: Bad Request

###### <span id="timestamps-to-rounds-400-schema"></span> Schema

##### <span id="timestamps-to-rounds-500"></span> 500
Status: Internal Server Error

###### <span id="timestamps-to-rounds-500-schema"></span> Schema

### <span id="total-stored-data"></span> Gets the total data currently storage used across all blobbers. (*total-stored-data*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/total-stored-data
```

# This endpoint returns the summation of all the Size fields in all the WriteMarkers sent to 0chain by blobbers

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#total-stored-data-200) | OK | StringMap |  | [schema](#total-stored-data-200-schema) |
| [400](#total-stored-data-400) | Bad Request |  |  | [schema](#total-stored-data-400-schema) |

#### Responses


##### <span id="total-stored-data-200"></span> 200 - StringMap
Status: OK

###### <span id="total-stored-data-200-schema"></span> Schema
   
  

[StringMap](#string-map)

##### <span id="total-stored-data-400"></span> 400
Status: Bad Request

###### <span id="total-stored-data-400-schema"></span> Schema

### <span id="transaction"></span> transaction (*transaction*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/transaction
```

Gets transaction information from transaction hash

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#transaction-200) | OK | Transaction |  | [schema](#transaction-200-schema) |
| [500](#transaction-500) | Internal Server Error |  |  | [schema](#transaction-500-schema) |

#### Responses


##### <span id="transaction-200"></span> 200 - Transaction
Status: OK

###### <span id="transaction-200-schema"></span> Schema
   
  

[Transaction](#transaction)

##### <span id="transaction-500"></span> 500
Status: Internal Server Error

###### <span id="transaction-500-schema"></span> Schema

### <span id="transaction-hashes"></span> transaction hashes (*transactionHashes*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/transactionHashes
```

Gets filtered list of transaction hashes from file information

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| content-hash | `query` | string | `string` |  |  |  | restrict to transactions by the specific content hash on write marker |
| limit | `query` | string | `string` |  |  |  | limit |
| look-up-hash | `query` | string | `string` |  |  |  | restrict to transactions by the specific look up hash on write marker |
| name | `query` | string | `string` |  |  |  | restrict to transactions by the specific file name on write marker |
| offset | `query` | string | `string` |  |  |  | offset |
| sort | `query` | string | `string` |  |  |  | desc or asc |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#transaction-hashes-200) | OK | stringArray |  | [schema](#transaction-hashes-200-schema) |
| [400](#transaction-hashes-400) | Bad Request |  |  | [schema](#transaction-hashes-400-schema) |
| [500](#transaction-hashes-500) | Internal Server Error |  |  | [schema](#transaction-hashes-500-schema) |

#### Responses


##### <span id="transaction-hashes-200"></span> 200 - stringArray
Status: OK

###### <span id="transaction-hashes-200-schema"></span> Schema
   
  


 [StringArray](#string-array)

##### <span id="transaction-hashes-400"></span> 400
Status: Bad Request

###### <span id="transaction-hashes-400-schema"></span> Schema

##### <span id="transaction-hashes-500"></span> 500
Status: Internal Server Error

###### <span id="transaction-hashes-500-schema"></span> Schema

### <span id="transactions"></span> transactions (*transactions*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/transactions
```

Gets filtered list of transaction information

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| block-end | `query` | string | `string` |  |  |  | restrict to transactions in specified start block and endblock |
| block-start | `query` | string | `string` |  |  |  | restrict to transactions in specified start block and endblock |
| block_hash | `query` | string | `string` |  |  |  | restrict to transactions in indicated block |
| client_id | `query` | string | `string` |  |  |  | restrict to transactions sent by the specified client |
| limit | `query` | string | `string` |  |  |  | limit |
| offset | `query` | string | `string` |  |  |  | offset |
| sort | `query` | string | `string` |  |  |  | desc or asc |
| to_client_id | `query` | string | `string` |  |  |  | restrict to transactions sent to a specified client |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#transactions-200) | OK | Transaction |  | [schema](#transactions-200-schema) |
| [400](#transactions-400) | Bad Request |  |  | [schema](#transactions-400-schema) |
| [500](#transactions-500) | Internal Server Error |  |  | [schema](#transactions-500-schema) |

#### Responses


##### <span id="transactions-200"></span> 200 - Transaction
Status: OK

###### <span id="transactions-200-schema"></span> Schema
   
  

[][Transaction](#transaction)

##### <span id="transactions-400"></span> 400
Status: Bad Request

###### <span id="transactions-400-schema"></span> Schema

##### <span id="transactions-500"></span> 500
Status: Internal Server Error

###### <span id="transactions-500-schema"></span> Schema

### <span id="vesting-config"></span> vesting config (*vesting_config*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/vesting_config
```

get vesting configuration settings

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#vesting-config-200) | OK | StringMap |  | [schema](#vesting-config-200-schema) |
| [500](#vesting-config-500) | Internal Server Error |  |  | [schema](#vesting-config-500-schema) |

#### Responses


##### <span id="vesting-config-200"></span> 200 - StringMap
Status: OK

###### <span id="vesting-config-200-schema"></span> Schema
   
  

[StringMap](#string-map)

##### <span id="vesting-config-500"></span> 500
Status: Internal Server Error

###### <span id="vesting-config-500-schema"></span> Schema

### <span id="writemarkers"></span> writemarkers (*writemarkers*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/writemarkers
```

Gets list of write markers satisfying filter

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| is_descending | `query` | string | `string` |  |  |  | is descending |
| limit | `query` | string | `string` |  |  |  | limit |
| offset | `query` | string | `string` |  |  |  | offset |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#writemarkers-200) | OK | WriteMarker |  | [schema](#writemarkers-200-schema) |
| [400](#writemarkers-400) | Bad Request |  |  | [schema](#writemarkers-400-schema) |
| [500](#writemarkers-500) | Internal Server Error |  |  | [schema](#writemarkers-500-schema) |

#### Responses


##### <span id="writemarkers-200"></span> 200 - WriteMarker
Status: OK

###### <span id="writemarkers-200-schema"></span> Schema
   
  

[][WriteMarker](#write-marker)

##### <span id="writemarkers-400"></span> 400
Status: Bad Request

###### <span id="writemarkers-400-schema"></span> Schema

##### <span id="writemarkers-500"></span> 500
Status: Internal Server Error

###### <span id="writemarkers-500-schema"></span> Schema

## Models

### <span id="allocation"></span> Allocation


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| AllocationID | string| `string` |  | |  |  |
| AllocationName | string| `string` |  | |  |  |
| Cancelled | boolean| `bool` |  | |  |  |
| CreatedAt | date-time (formatted string)| `strfmt.DateTime` |  | |  |  |
| DataShards | int64 (formatted integer)| `int64` |  | |  |  |
| DeletedAt | [DeletedAt](#deleted-at)| `DeletedAt` |  | |  |  |
| Expiration | int64 (formatted integer)| `int64` |  | |  |  |
| FailedChallenges | int64 (formatted integer)| `int64` |  | |  |  |
| Finalized | boolean| `bool` |  | |  |  |
| ID | uint64 (formatted integer)| `uint64` |  | |  |  |
| IsImmutable | boolean| `bool` |  | |  |  |
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
| AllocationID | string| `string` |  | |  |  |
| BlobberID | string| `string` |  | |  |  |
| CreatedAt | date-time (formatted string)| `strfmt.DateTime` |  | |  |  |
| DeletedAt | [DeletedAt](#deleted-at)| `DeletedAt` |  | |  |  |
| ID | uint64 (formatted integer)| `uint64` |  | |  |  |
| MinLockDemand | double (formatted number)| `float64` |  | |  |  |
| ReadPrice | int64 (formatted integer)| `int64` |  | |  |  |
| UpdatedAt | date-time (formatted string)| `strfmt.DateTime` |  | |  |  |
| WritePrice | int64 (formatted integer)| `int64` |  | |  |  |
| max_offer_duration | [Duration](#duration)| `Duration` |  | |  |  |



### <span id="approved-minter"></span> ApprovedMinter


  

| Name | Type | Go type | Default | Description | Example |
|------|------|---------| ------- |-------------|---------|
| ApprovedMinter | int64 (formatted integer)| int64 | |  |  |



### <span id="blobber-allocation"></span> BlobberAllocation


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| AllocationID | string| `string` |  | |  |  |
| AllocationRoot | string| `string` |  | |  |  |
| BlobberID | string| `string` |  | |  |  |
| Size | int64 (formatted integer)| `int64` |  | | Size is blobber allocation maximum size |  |
| blobber_allocs_partition_loc | [PartitionLocation](#partition-location)| `PartitionLocation` |  | |  |  |
| challenge_pool_integral_value | [Coin](#coin)| `Coin` |  | |  |  |
| challenge_reward | [Coin](#coin)| `Coin` |  | |  |  |
| min_lock_demand | [Coin](#coin)| `Coin` |  | |  |  |
| penalty | [Coin](#coin)| `Coin` |  | |  |  |
| read_reward | [Coin](#coin)| `Coin` |  | |  |  |
| returned | [Coin](#coin)| `Coin` |  | |  |  |
| spent | [Coin](#coin)| `Coin` |  | |  |  |
| stats | [StorageAllocationStats](#storage-allocation-stats)| `StorageAllocationStats` |  | |  |  |
| terms | [Terms](#terms)| `Terms` |  | |  |  |
| write_marker | [WriteMarker](#write-marker)| `WriteMarker` |  | |  |  |



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
| InactiveRounds | int64 (formatted integer)| `int64` |  | |  |  |
| OpenChallenges | uint64 (formatted integer)| `uint64` |  | |  |  |
| RankMetric | double (formatted number)| `float64` |  | |  |  |
| ReadData | int64 (formatted integer)| `int64` |  | |  |  |
| SavedData | int64 (formatted integer)| `int64` |  | |  |  |
| offers_total | [Coin](#coin)| `Coin` |  | |  |  |
| total_service_charge | [Coin](#coin)| `Coin` |  | |  |  |
| total_stake | [Coin](#coin)| `Coin` |  | |  |  |
| unstake_total | [Coin](#coin)| `Coin` |  | |  |  |
| write_price | [Coin](#coin)| `Coin` |  | |  |  |



### <span id="block"></span> Block


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| ChainId | string| `string` |  | |  |  |
| CreatedAt | date-time (formatted string)| `strfmt.DateTime` |  | |  |  |
| CreatedAt | date-time (formatted string)| `strfmt.DateTime` |  | |  |  |
| CreationDate | int64 (formatted integer)| `int64` |  | |  |  |
| DeletedAt | [DeletedAt](#deleted-at)| `DeletedAt` |  | |  |  |
| Hash | string| `string` |  | |  |  |
| ID | uint64 (formatted integer)| `uint64` |  | |  |  |
| IsFinalised | boolean| `bool` |  | |  |  |
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
| StateHash | string| `string` |  | |  |  |
| UpdatedAt | date-time (formatted string)| `strfmt.DateTime` |  | |  |  |
| Version | string| `string` |  | |  |  |



### <span id="challenge"></span> Challenge


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| AllocationID | string| `string` |  | |  |  |
| AllocationRoot | string| `string` |  | |  |  |
| BlobberID | string| `string` |  | |  |  |
| ChallengeID | string| `string` |  | |  |  |
| CreatedAt | date-time (formatted string)| `strfmt.DateTime` |  | |  |  |
| DeletedAt | [DeletedAt](#deleted-at)| `DeletedAt` |  | |  |  |
| ExpiredN | int64 (formatted integer)| `int64` |  | |  |  |
| ID | uint64 (formatted integer)| `uint64` |  | |  |  |
| Passed | boolean| `bool` |  | |  |  |
| Responded | boolean| `bool` |  | |  |  |
| RoundResponded | int64 (formatted integer)| `int64` |  | |  |  |
| Seed | int64 (formatted integer)| `int64` |  | |  |  |
| UpdatedAt | date-time (formatted string)| `strfmt.DateTime` |  | |  |  |
| ValidatorsID | string| `string` |  | |  |  |
| created_at | [Timestamp](#timestamp)| `Timestamp` |  | |  |  |



### <span id="challenges"></span> Challenges


  

[][Challenge](#challenge)

### <span id="challenges-response"></span> ChallengesResponse


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| BlobberID | string| `string` |  | |  |  |
| Challenges | [][StorageChallengeResponse](#storage-challenge-response)| `[]*StorageChallengeResponse` |  | |  |  |



### <span id="client"></span> Client


> go:generate msgp -io=false -tests=false -v
Client - data structure that holds the client data
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| ID | string| `string` |  | |  |  |
| PublicKey | string| `string` |  | |  |  |
| Version | string| `string` |  | |  |  |
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



### <span id="delegate-pool-stat"></span> DelegatePoolStat


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| DelegateID | string| `string` |  | |  |  |
| ID | string| `string` |  | |  |  |
| ProviderId | string| `string` |  | |  |  |
| ProviderType | int64 (formatted integer)| `int64` |  | |  |  |
| RoundCreated | int64 (formatted integer)| `int64` |  | |  |  |
| Status | string| `string` |  | |  |  |
| UnStake | boolean| `bool` |  | |  |  |
| balance | [Coin](#coin)| `Coin` |  | |  |  |
| rewards | [Coin](#coin)| `Coin` |  | |  |  |
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
| Duration | int64 (formatted integer)| int64 | | A Duration represents the elapsed time between two instants
as an int64 nanosecond count. The representation limits the
largest representable duration to approximately 290 years. |  |



### <span id="error"></span> Error


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| CreatedAt | date-time (formatted string)| `strfmt.DateTime` |  | |  |  |
| DeletedAt | [DeletedAt](#deleted-at)| `DeletedAt` |  | |  |  |
| Error | string| `string` |  | |  |  |
| ID | uint64 (formatted integer)| `uint64` |  | |  |  |
| TransactionID | string| `string` |  | |  |  |
| UpdatedAt | date-time (formatted string)| `strfmt.DateTime` |  | |  |  |



### <span id="event"></span> Event


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| BlockNumber | int64 (formatted integer)| `int64` |  | |  |  |
| CreatedAt | date-time (formatted string)| `strfmt.DateTime` |  | |  |  |
| Data | [interface{}](#interface)| `interface{}` |  | |  |  |
| DeletedAt | [DeletedAt](#deleted-at)| `DeletedAt` |  | |  |  |
| ID | uint64 (formatted integer)| `uint64` |  | |  |  |
| Index | string| `string` |  | |  |  |
| TxHash | string| `string` |  | |  |  |
| UpdatedAt | date-time (formatted string)| `strfmt.DateTime` |  | |  |  |
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


  

[interface{}](#interface)

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



### <span id="miner-geolocation"></span> MinerGeolocation


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| Latitude | double (formatted number)| `float64` |  | |  |  |
| Longitude | double (formatted number)| `float64` |  | |  |  |
| MinerID | string| `string` |  | |  |  |



### <span id="miner-node"></span> MinerNode


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| BuildTag | string| `string` |  | |  |  |
| Delete | boolean| `bool` |  | |  |  |
| Host | string| `string` |  | |  |  |
| ID | string| `string` |  | |  |  |
| LastSettingUpdateRound | int64 (formatted integer)| `int64` |  | | LastSettingUpdateRound will be set to round number when settings were updated |  |
| N2NHost | string| `string` |  | |  |  |
| Path | string| `string` |  | |  |  |
| Pools | map of [DelegatePool](#delegate-pool)| `map[string]DelegatePool` |  | |  |  |
| Port | int64 (formatted integer)| `int64` |  | |  |  |
| PublicKey | string| `string` |  | |  |  |
| ShortName | string| `string` |  | |  |  |
| geolocation | [SimpleNodeGeolocation](#simple-node-geolocation)| `SimpleNodeGeolocation` |  | |  |  |
| last_health_check | [Timestamp](#timestamp)| `Timestamp` |  | |  |  |
| minter | [ApprovedMinter](#approved-minter)| `ApprovedMinter` |  | |  |  |
| node_type | [NodeType](#node-type)| `NodeType` |  | |  |  |
| rewards | [Coin](#coin)| `Coin` |  | |  |  |
| settings | [Settings](#settings)| `Settings` |  | |  |  |
| total_stake | [Coin](#coin)| `Coin` |  | |  |  |



### <span id="miner-nodes"></span> MinerNodes


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| Nodes | [][MinerNode](#miner-node)| `[]*MinerNode` |  | |  |  |



### <span id="model"></span> Model


> Model a basic GoLang struct which includes the following fields: ID, CreatedAt, UpdatedAt, DeletedAt
It may be embedded into your model or you may build your own model without it
type User struct {
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
| PublicKey | string| `string` |  | |  |  |
| SetIndex | int64 (formatted integer)| `int64` |  | |  |  |
| Status | int64 (formatted integer)| `int64` |  | |  |  |
| Version | string| `string` |  | |  |  |
| creation_date | [Timestamp](#timestamp)| `Timestamp` |  | |  |  |
| info | [Info](#info)| `Info` |  | |  |  |
| type | [NodeType](#node-type)| `NodeType` |  | |  |  |



### <span id="node-type"></span> NodeType


  

| Name | Type | Go type | Default | Description | Example |
|------|------|---------| ------- |-------------|---------|
| NodeType | int8 (formatted integer)| int8 | |  |  |



### <span id="null-time"></span> NullTime


> NullTime implements the Scanner interface so
it can be used as a scan destination, similar to NullString.
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| Time | date-time (formatted string)| `strfmt.DateTime` |  | |  |  |
| Valid | boolean| `bool` |  | |  |  |



### <span id="partition-location"></span> PartitionLocation


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| Location | int64 (formatted integer)| `int64` |  | |  |  |
| Timestamp | [Timestamp](#timestamp)| `Timestamp` |  | |  |  |



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



### <span id="price-range"></span> PriceRange


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| max | [Coin](#coin)| `Coin` |  | |  |  |
| min | [Coin](#coin)| `Coin` |  | |  |  |



### <span id="provider"></span> Provider


  

| Name | Type | Go type | Default | Description | Example |
|------|------|---------| ------- |-------------|---------|
| Provider | int64 (formatted integer)| int64 | |  |  |



### <span id="provider-rewards"></span> ProviderRewards


> ProviderRewards is a tables stores the rewards and total_rewards for all kinds of providers
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| CreatedAt | date-time (formatted string)| `strfmt.DateTime` |  | |  |  |
| DeletedAt | [DeletedAt](#deleted-at)| `DeletedAt` |  | |  |  |
| ID | uint64 (formatted integer)| `uint64` |  | |  |  |
| ProviderID | string| `string` |  | |  |  |
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
| DeletedAt | [DeletedAt](#deleted-at)| `DeletedAt` |  | |  |  |
| ID | uint64 (formatted integer)| `uint64` |  | |  |  |
| Owner | [User](#user)| `User` |  | |  |  |
| OwnerID | string| `string` |  | |  |  |
| PayerID | string| `string` |  | |  |  |
| ReadCounter | int64 (formatted integer)| `int64` |  | |  |  |
| ReadSize | double (formatted number)| `float64` |  | |  |  |
| Signature | string| `string` |  | |  |  |
| Timestamp | int64 (formatted integer)| `int64` |  | |  |  |
| TransactionID | string| `string` |  | |  |  |
| UpdatedAt | date-time (formatted string)| `strfmt.DateTime` |  | |  |  |
| User | [User](#user)| `User` |  | |  |  |



### <span id="reward-partition-location"></span> RewardPartitionLocation


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| Index | int64 (formatted integer)| `int64` |  | |  |  |
| StartRound | int64 (formatted integer)| `int64` |  | |  |  |
| timestamp | [Timestamp](#timestamp)| `Timestamp` |  | |  |  |



### <span id="settings"></span> Settings


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| DelegateWallet | string| `string` |  | |  |  |
| MaxNumDelegates | int64 (formatted integer)| `int64` |  | |  |  |
| ServiceChargeRatio | double (formatted number)| `float64` |  | |  |  |
| max_stake | [Coin](#coin)| `Coin` |  | |  |  |
| min_stake | [Coin](#coin)| `Coin` |  | |  |  |



### <span id="sharder-geolocation"></span> SharderGeolocation


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| Latitude | double (formatted number)| `float64` |  | |  |  |
| Longitude | double (formatted number)| `float64` |  | |  |  |
| SharderID | string| `string` |  | |  |  |



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
| Host | string| `string` |  | |  |  |
| ID | string| `string` |  | |  |  |
| LastSettingUpdateRound | int64 (formatted integer)| `int64` |  | | LastSettingUpdateRound will be set to round number when settings were updated |  |
| N2NHost | string| `string` |  | |  |  |
| Path | string| `string` |  | |  |  |
| Port | int64 (formatted integer)| `int64` |  | |  |  |
| PublicKey | string| `string` |  | |  |  |
| ShortName | string| `string` |  | |  |  |
| geolocation | [SimpleNodeGeolocation](#simple-node-geolocation)| `SimpleNodeGeolocation` |  | |  |  |
| last_health_check | [Timestamp](#timestamp)| `Timestamp` |  | |  |  |
| node_type | [NodeType](#node-type)| `NodeType` |  | |  |  |
| total_stake | [Coin](#coin)| `Coin` |  | |  |  |



### <span id="simple-node-geolocation"></span> SimpleNodeGeolocation


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| Latitude | double (formatted number)| `float64` |  | |  |  |
| Longitude | double (formatted number)| `float64` |  | |  |  |



### <span id="simple-nodes"></span> SimpleNodes


> not thread safe
  



[SimpleNodes](#simple-nodes)

### <span id="stake-pool"></span> StakePool


> StakePool holds delegate information for an 0chain providers
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| Pools | map of [DelegatePool](#delegate-pool)| `map[string]DelegatePool` |  | |  |  |
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
| unstake_total | [Coin](#coin)| `Coin` |  | |  |  |



### <span id="storage-allocation"></span> StorageAllocation


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| BlobberAllocs | [][BlobberAllocation](#blobber-allocation)| `[]*BlobberAllocation` |  | | Blobbers not to be used anywhere except /allocation and /allocations table
if Blobbers are getting used in any smart-contract, we should avoid. |  |
| Canceled | boolean| `bool` |  | | Canceled set to true where allocation finalized by cancel_allocation
transaction. |  |
| Curators | []string| `[]string` |  | |  |  |
| DataShards | int64 (formatted integer)| `int64` |  | |  |  |
| DiverseBlobbers | boolean| `bool` |  | |  |  |
| FileOptions | uint8 (formatted integer)| `uint8` |  | | FileOptions to define file restrictions on an allocation for third-parties
default 00000000 for all crud operations suggesting only owner has the below listed abilities.
enabling option/s allows any third party to perform certain ops
00000001 - 1  - upload
00000010 - 2  - delete
00000100 - 4  - update
00001000 - 8  - move
00010000 - 16 - copy
00100000 - 32 - rename |  |
| Finalized | boolean| `bool` |  | | Finalized is true where allocation has been finalized. |  |
| ID | string| `string` |  | | ID is unique allocation ID that is equal to hash of transaction with
which the allocation has created. |  |
| IsImmutable | boolean| `bool` |  | | Defines mutability of the files in the allocation, used by blobber on CommitWrite |  |
| Name | string| `string` |  | | Name is the name of an allocation |  |
| Owner | string| `string` |  | |  |  |
| OwnerPublicKey | string| `string` |  | |  |  |
| ParityShards | int64 (formatted integer)| `int64` |  | |  |  |
| PreferredBlobbers | []string| `[]string` |  | |  |  |
| Size | int64 (formatted integer)| `int64` |  | |  |  |
| ThirdPartyExtendable | boolean| `bool` |  | | Flag to determine if anyone can extend this allocation |  |
| Tx | string| `string` |  | | Tx keeps hash with which the allocation has created or updated. todo do we need this field? |  |
| expiration_date | [Timestamp](#timestamp)| `Timestamp` |  | |  |  |
| moved_back | [Coin](#coin)| `Coin` |  | |  |  |
| moved_to_challenge | [Coin](#coin)| `Coin` |  | |  |  |
| moved_to_validators | [Coin](#coin)| `Coin` |  | |  |  |
| read_price_range | [PriceRange](#price-range)| `PriceRange` |  | |  |  |
| start_time | [Timestamp](#timestamp)| `Timestamp` |  | |  |  |
| stats | [StorageAllocationStats](#storage-allocation-stats)| `StorageAllocationStats` |  | |  |  |
| time_unit | [Duration](#duration)| `Duration` |  | |  |  |
| write_pool | [Coin](#coin)| `Coin` |  | |  |  |
| write_price_range | [PriceRange](#price-range)| `PriceRange` |  | |  |  |



### <span id="storage-allocation-stats"></span> StorageAllocationStats


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| FailedChallenges | int64 (formatted integer)| `int64` |  | |  |  |
| LastestClosedChallengeTxn | string| `string` |  | |  |  |
| NumReads | int64 (formatted integer)| `int64` |  | |  |  |
| NumWrites | int64 (formatted integer)| `int64` |  | |  |  |
| OpenChallenges | int64 (formatted integer)| `int64` |  | |  |  |
| SuccessChallenges | int64 (formatted integer)| `int64` |  | |  |  |
| TotalChallenges | int64 (formatted integer)| `int64` |  | |  |  |
| UsedSize | int64 (formatted integer)| `int64` |  | |  |  |



### <span id="storage-challenge"></span> StorageChallenge


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| AllocationID | string| `string` |  | |  |  |
| BlobberID | string| `string` |  | |  |  |
| ID | string| `string` |  | |  |  |
| Responded | boolean| `bool` |  | |  |  |
| TotalValidators | int64 (formatted integer)| `int64` |  | |  |  |
| ValidatorIDs | []string| `[]string` |  | |  |  |
| created | [Timestamp](#timestamp)| `Timestamp` |  | |  |  |



### <span id="storage-challenge-response"></span> StorageChallengeResponse


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| AllocationID | string| `string` |  | |  |  |
| AllocationRoot | string| `string` |  | |  |  |
| BlobberID | string| `string` |  | |  |  |
| ID | string| `string` |  | |  |  |
| Responded | boolean| `bool` |  | |  |  |
| Seed | int64 (formatted integer)| `int64` |  | |  |  |
| TotalValidators | int64 (formatted integer)| `int64` |  | |  |  |
| ValidatorIDs | []string| `[]string` |  | |  |  |
| Validators | [][ValidationNode](#validation-node)| `[]*ValidationNode` |  | |  |  |
| created | [Timestamp](#timestamp)| `Timestamp` |  | |  |  |



### <span id="storage-node"></span> StorageNode


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| Allocated | int64 (formatted integer)| `int64` |  | |  |  |
| BaseURL | string| `string` |  | |  |  |
| Capacity | int64 (formatted integer)| `int64` |  | |  |  |
| DataReadLastRewardRound | double (formatted number)| `float64` |  | |  |  |
| ID | string| `string` |  | |  |  |
| LastRewardDataReadRound | int64 (formatted integer)| `int64` |  | |  |  |
| SavedData | int64 (formatted integer)| `int64` |  | |  |  |
| geolocation | [StorageNodeGeolocation](#storage-node-geolocation)| `StorageNodeGeolocation` |  | |  |  |
| last_health_check | [Timestamp](#timestamp)| `Timestamp` |  | |  |  |
| provider_type | [Provider](#provider)| `Provider` |  | |  |  |
| reward_partition | [RewardPartitionLocation](#reward-partition-location)| `RewardPartitionLocation` |  | |  |  |
| stake_pool_settings | [Settings](#settings)| `Settings` |  | |  |  |
| terms | [Terms](#terms)| `Terms` |  | |  |  |



### <span id="storage-node-geolocation"></span> StorageNodeGeolocation


> Move to the core, in case of multi-entity use of geo data
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| Latitude | double (formatted number)| `float64` |  | |  |  |
| Longitude | double (formatted number)| `float64` |  | |  |  |



### <span id="string-map"></span> StringMap


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| Fields | map of string| `map[string]string` |  | |  |  |



### <span id="terms"></span> Terms


> but any existing offer will use terms of offer signing time.
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| MinLockDemand | double (formatted number)| `float64` |  | | MinLockDemand in number in [0; 1] range. It represents part of
allocation should be locked for the blobber rewards even if
user never write something to the blobber. |  |
| max_offer_duration | [Duration](#duration)| `Duration` |  | |  |  |
| read_price | [Coin](#coin)| `Coin` |  | |  |  |
| write_price | [Coin](#coin)| `Coin` |  | |  |  |



### <span id="timestamp"></span> Timestamp


> go:generate msgp -io=false -tests=false -v
Timestamp - just a wrapper to control the json encoding */
  



| Name | Type | Go type | Default | Description | Example |
|------|------|---------| ------- |-------------|---------|
| Timestamp | int64 (formatted integer)| int64 | | go:generate msgp -io=false -tests=false -v
Timestamp - just a wrapper to control the json encoding */ |  |



### <span id="transaction"></span> Transaction


> Transaction model to save the transaction data
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| BlockHash | string| `string` |  | |  |  |
| ClientId | string| `string` |  | |  |  |
| CreatedAt | date-time (formatted string)| `strfmt.DateTime` |  | |  |  |
| CreationDate | int64 (formatted integer)| `int64` |  | |  |  |
| DeletedAt | [DeletedAt](#deleted-at)| `DeletedAt` |  | |  |  |
| Hash | string| `string` |  | |  |  |
| ID | uint64 (formatted integer)| `uint64` |  | |  |  |
| Nonce | int64 (formatted integer)| `int64` |  | |  |  |
| OutputHash | string| `string` |  | |  |  |
| Round | int64 (formatted integer)| `int64` |  | |  |  |
| Signature | string| `string` |  | |  |  |
| Status | int64 (formatted integer)| `int64` |  | |  |  |
| ToClientId | string| `string` |  | |  |  |
| TransactionData | string| `string` |  | |  |  |
| TransactionOutput | string| `string` |  | |  |  |
| TransactionType | int64 (formatted integer)| `int64` |  | |  |  |
| UpdatedAt | date-time (formatted string)| `strfmt.DateTime` |  | |  |  |
| Version | string| `string` |  | |  |  |
| fee | [Coin](#coin)| `Coin` |  | |  |  |
| value | [Coin](#coin)| `Coin` |  | |  |  |



### <span id="user"></span> User


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| CreatedAt | date-time (formatted string)| `strfmt.DateTime` |  | |  |  |
| DeletedAt | [DeletedAt](#deleted-at)| `DeletedAt` |  | |  |  |
| ID | uint64 (formatted integer)| `uint64` |  | |  |  |
| Nonce | int64 (formatted integer)| `int64` |  | |  |  |
| Round | int64 (formatted integer)| `int64` |  | |  |  |
| TxnHash | string| `string` |  | |  |  |
| UpdatedAt | date-time (formatted string)| `strfmt.DateTime` |  | |  |  |
| UserID | string| `string` |  | |  |  |
| balance | [Coin](#coin)| `Coin` |  | |  |  |
| change | [Coin](#coin)| `Coin` |  | |  |  |



### <span id="user-pool-stat"></span> UserPoolStat


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| Pools | map of [[]*DelegatePoolStat](#delegate-pool-stat)| `map[string][]DelegatePoolStat` |  | |  |  |



### <span id="validation-node"></span> ValidationNode


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| BaseURL | string| `string` |  | |  |  |
| ID | string| `string` |  | |  |  |
| provider_type | [Provider](#provider)| `Provider` |  | |  |  |
| stake_pool_settings | [Settings](#settings)| `Settings` |  | |  |  |



### <span id="validator"></span> Validator


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| BaseUrl | string| `string` |  | |  |  |
| CreatedAt | date-time (formatted string)| `strfmt.DateTime` |  | |  |  |
| DelegateWallet | string| `string` |  | |  |  |
| ID | string| `string` |  | |  |  |
| NumDelegates | int64 (formatted integer)| `int64` |  | |  |  |
| PublicKey | string| `string` |  | |  |  |
| ServiceCharge | double (formatted number)| `float64` |  | |  |  |
| UpdatedAt | date-time (formatted string)| `strfmt.DateTime` |  | |  |  |
| max_stake | [Coin](#coin)| `Coin` |  | |  |  |
| min_stake | [Coin](#coin)| `Coin` |  | |  |  |
| rewards | [ProviderRewards](#provider-rewards)| `ProviderRewards` |  | |  |  |
| total_stake | [Coin](#coin)| `Coin` |  | |  |  |
| unstake_total | [Coin](#coin)| `Coin` |  | |  |  |



### <span id="version-field"></span> VersionField


> go:generate msgp -io=false -tests=false -v
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| Version | string| `string` |  | |  |  |



### <span id="write-marker"></span> WriteMarker


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| AllocationID | string| `string` |  | |  |  |
| AllocationRoot | string| `string` |  | |  |  |
| BlobberID | string| `string` |  | |  |  |
| ClientID | string| `string` |  | |  |  |
| ContentHash | string| `string` |  | |  |  |
| LookupHash | string| `string` |  | | file info |  |
| Name | string| `string` |  | |  |  |
| Operation | string| `string` |  | |  |  |
| PreviousAllocationRoot | string| `string` |  | |  |  |
| Signature | string| `string` |  | |  |  |
| Size | int64 (formatted integer)| `int64` |  | |  |  |
| Timestamp | [Timestamp](#timestamp)| `Timestamp` |  | |  |  |



### <span id="authorizer-node"></span> authorizerNode


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| ID | string| `string` |  | |  |  |
| URL | string| `string` |  | |  |  |



### <span id="authorizer-nodes-response"></span> authorizerNodesResponse


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| Nodes | [][AuthorizerNode](#authorizer-node)| `[]*AuthorizerNode` |  | |  |  |



### <span id="authorizer-response"></span> authorizerResponse


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| AuthorizerID | string| `string` |  | |  |  |
| DelegateWallet | string| `string` |  | | stake_pool_settings |  |
| LastHealthCheck | int64 (formatted integer)| `int64` |  | | Stats |  |
| Latitude | double (formatted number)| `float64` |  | | Geolocation |  |
| Longitude | double (formatted number)| `float64` |  | |  |  |
| NumDelegates | int64 (formatted integer)| `int64` |  | |  |  |
| ServiceCharge | double (formatted number)| `float64` |  | |  |  |
| URL | string| `string` |  | |  |  |
| fee | [Coin](#coin)| `Coin` |  | |  |  |
| max_stake | [Coin](#coin)| `Coin` |  | |  |  |
| min_stake | [Coin](#coin)| `Coin` |  | |  |  |



### <span id="challenge-pool-stat"></span> challengePoolStat


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| Finalized | boolean| `bool` |  | |  |  |
| ID | string| `string` |  | |  |  |
| balance | [Coin](#coin)| `Coin` |  | |  |  |
| expiration | [Timestamp](#timestamp)| `Timestamp` |  | |  |  |
| start_time | [Timestamp](#timestamp)| `Timestamp` |  | |  |  |



### <span id="client-pools"></span> clientPools


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| Pools | []string| `[]string` |  | |  |  |



### <span id="dest-info"></span> destInfo


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| ID | string| `string` |  | |  |  |
| earned | [Coin](#coin)| `Coin` |  | |  |  |
| last | [Timestamp](#timestamp)| `Timestamp` |  | |  |  |
| vested | [Coin](#coin)| `Coin` |  | |  |  |
| wanted | [Coin](#coin)| `Coin` |  | |  |  |



### <span id="event-list"></span> eventList


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| Events | [][Event](#event)| `[]*Event` |  | |  |  |



### <span id="full-block"></span> fullBlock


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| ChainId | string| `string` |  | |  |  |
| CreatedAt | date-time (formatted string)| `strfmt.DateTime` |  | |  |  |
| CreatedAt | date-time (formatted string)| `strfmt.DateTime` |  | |  |  |
| CreationDate | int64 (formatted integer)| `int64` |  | |  |  |
| DeletedAt | [DeletedAt](#deleted-at)| `DeletedAt` |  | |  |  |
| Hash | string| `string` |  | |  |  |
| ID | uint64 (formatted integer)| `uint64` |  | |  |  |
| IsFinalised | boolean| `bool` |  | |  |  |
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
| StateHash | string| `string` |  | |  |  |
| Transactions | [][Transaction](#transaction)| `[]*Transaction` |  | |  |  |
| UpdatedAt | date-time (formatted string)| `strfmt.DateTime` |  | |  |  |
| Version | string| `string` |  | |  |  |



### <span id="info"></span> info


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| ClientID | string| `string` |  | |  |  |
| Description | string| `string` |  | |  |  |
| Destinations | [][DestInfo](#dest-info)| `[]*DestInfo` |  | |  |  |
| ID | string| `string` |  | |  |  |
| balance | [Coin](#coin)| `Coin` |  | |  |  |
| expire_at | [Timestamp](#timestamp)| `Timestamp` |  | |  |  |
| left | [Coin](#coin)| `Coin` |  | |  |  |
| start_time | [Timestamp](#timestamp)| `Timestamp` |  | |  |  |



### <span id="node-stat"></span> nodeStat


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| BuildTag | string| `string` |  | |  |  |
| Delete | boolean| `bool` |  | |  |  |
| Host | string| `string` |  | |  |  |
| ID | string| `string` |  | |  |  |
| LastSettingUpdateRound | int64 (formatted integer)| `int64` |  | | LastSettingUpdateRound will be set to round number when settings were updated |  |
| N2NHost | string| `string` |  | |  |  |
| Path | string| `string` |  | |  |  |
| Pools | map of [DelegatePool](#delegate-pool)| `map[string]DelegatePool` |  | |  |  |
| Port | int64 (formatted integer)| `int64` |  | |  |  |
| PublicKey | string| `string` |  | |  |  |
| Round | int64 (formatted integer)| `int64` |  | |  |  |
| ShortName | string| `string` |  | |  |  |
| TotalReward | int64 (formatted integer)| `int64` |  | |  |  |
| geolocation | [SimpleNodeGeolocation](#simple-node-geolocation)| `SimpleNodeGeolocation` |  | |  |  |
| last_health_check | [Timestamp](#timestamp)| `Timestamp` |  | |  |  |
| minter | [ApprovedMinter](#approved-minter)| `ApprovedMinter` |  | |  |  |
| node_type | [NodeType](#node-type)| `NodeType` |  | |  |  |
| rewards | [Coin](#coin)| `Coin` |  | |  |  |
| settings | [Settings](#settings)| `Settings` |  | |  |  |
| total_stake | [Coin](#coin)| `Coin` |  | |  |  |



### <span id="periodic-response"></span> periodicResponse


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| Restart | string| `string` |  | |  |  |
| Start | date-time (formatted string)| `strfmt.DateTime` |  | |  |  |
| tokens_allowed | [Coin](#coin)| `Coin` |  | |  |  |
| tokens_poured | [Coin](#coin)| `Coin` |  | |  |  |



### <span id="read-markers-count"></span> readMarkersCount


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| ReadMarkersCount | int64 (formatted integer)| `int64` |  | |  |  |



### <span id="read-pool"></span> readPool


> one for the allocations that the client (client_id) owns
and the other for the allocations that the client (client_id) doesn't own
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| balance | [Coin](#coin)| `Coin` |  | |  |  |



### <span id="storage-node-response"></span> storageNodeResponse


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| Allocated | int64 (formatted integer)| `int64` |  | |  |  |
| BaseURL | string| `string` |  | |  |  |
| Capacity | int64 (formatted integer)| `int64` |  | |  |  |
| CreationRound | int64 (formatted integer)| `int64` |  | |  |  |
| DataReadLastRewardRound | double (formatted number)| `float64` |  | |  |  |
| ID | string| `string` |  | |  |  |
| LastRewardDataReadRound | int64 (formatted integer)| `int64` |  | |  |  |
| ReadData | int64 (formatted integer)| `int64` |  | |  |  |
| SavedData | int64 (formatted integer)| `int64` |  | |  |  |
| UsedAllocation | int64 (formatted integer)| `int64` |  | |  |  |
| geolocation | [StorageNodeGeolocation](#storage-node-geolocation)| `StorageNodeGeolocation` |  | |  |  |
| last_health_check | [Timestamp](#timestamp)| `Timestamp` |  | |  |  |
| provider_type | [Provider](#provider)| `Provider` |  | |  |  |
| reward_partition | [RewardPartitionLocation](#reward-partition-location)| `RewardPartitionLocation` |  | |  |  |
| stake_pool_settings | [Settings](#settings)| `Settings` |  | |  |  |
| terms | [Terms](#terms)| `Terms` |  | |  |  |
| total_offers | [Coin](#coin)| `Coin` |  | |  |  |
| total_service_charge | [Coin](#coin)| `Coin` |  | |  |  |
| total_stake | [Coin](#coin)| `Coin` |  | |  |  |
| uncollected_service_charge | [Coin](#coin)| `Coin` |  | |  |  |



### <span id="storage-nodes-response"></span> storageNodesResponse


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| Nodes | [][StorageNodeResponse](#storage-node-response)| `[]*StorageNodeResponse` |  | |  |  |



### <span id="string-array"></span> stringArray


  

[]string

### <span id="timestamp-to-round-resp"></span> timestampToRoundResp


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| Rounds | []int64 (formatted integer)| `[]int64` |  | |  |  |

### <span id="user-locked-total-response"></span> userLockedTotalResponse


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| Total | int64 (formatted integer)| `int64` |  | |  |  |