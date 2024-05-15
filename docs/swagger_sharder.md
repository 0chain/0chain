


# Sharder Smart Contract API:
  

## Informations

### Version

0.1.0

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
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/alloc-blobber-term | [get alloc blobber terms](#get-alloc-blobber-terms) | Get terms of storage service for a specific allocation and blobber (write_price, read_price) if blobber_id is specified, otherwise, get terms of service for all blobbers of the allocation. |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d3/GetGlobalConfig | [get global config](#get-global-config) |  |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/alloc_blobbers | [alloc blobbers](#alloc-blobbers) | returns list of all active blobbers that match the allocation request, or an error if not enough blobbers are available. |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/alloc_write_marker_count | [alloc write marker count](#alloc-write-marker-count) |  |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/allocation | [allocation](#allocation) |  |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/allocation-update-min-lock | [allocation update min lock](#allocation-update-min-lock) | Calculates the cost for updating an allocation. |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/allocations | [allocations](#allocations) |  |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/blobber-allocations | [blobber allocations](#blobber-allocations) |  |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/blobber-challenges | [blobber challenges](#blobber-challenges) |  |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/blobber_ids | [blobber ids](#blobber-ids) |  |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/block | [block](#block) |  |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/collected_reward | [collected reward](#collected-reward) | Returns collected reward for a client_id. |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/configs | [configs](#configs) |  |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/count_readmarkers | [count readmarkers](#count-readmarkers) |  |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/delegate-rewards | [delegate rewards](#delegate-rewards) |  |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/errors | [errors](#errors) |  |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d3/faucet_config | [faucet config](#faucet-config) |  |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/free_alloc_blobbers | [free alloc blobbers](#free-alloc-blobbers) | returns list of all blobbers alive that match the free allocation request. |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d3/getAuthorizer | [get authorizer](#get-authorizer) |  |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712e0/getAuthorizerNodes | [get authorizer nodes](#get-authorizer-nodes) |  |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/getBlobber | [get blobber](#get-blobber) |  |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/getchallenge | [get challenge](#get-challenge) |  |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/getChallengePoolStat | [get challenge pool stat](#get-challenge-pool-stat) |  |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/getClientPools | [get client pools](#get-client-pools) |  |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/getDkgList | [get dkg list](#get-dkg-list) |  |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/getEvents | [get events](#get-events) |  |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/getExpiredAllocations | [get expired allocations](#get-expired-allocations) | Get expired allocations for a specific blobber. Retrieves a list of expired allocations associated with a specified blobber. |
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
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/getUserPools | [get user pools](#get-user-pools) |  |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/getUserStakePoolStat | [get user stake pool stat](#get-user-stake-pool-stat) |  |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/getWriteMarkers | [get write markers](#get-write-markers) |  |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/get_blocks | [get blocks](#get-blocks) | Gets block information for all blocks. Todo: We need to add a filter to this. |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/get_miners_stats | [get miners stats](#get-miners-stats) |  |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/get_sharders_stats | [get sharders stats](#get-sharders-stats) |  |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/get_validator | [get validator](#get-validator) |  |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/getblobbers | [getblobbers](#getblobbers) | Gets list of all blobbers alive (e.g. excluding blobbers with zero capacity). |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d3/globalPeriodicLimit | [global periodic limit](#global-periodic-limit) |  |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/globalSettings | [global settings](#global-settings) |  |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/hardfork | [hardfork](#hardfork) |  |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/latestreadmarker | [latestreadmarker](#latestreadmarker) |  |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/nodePoolStat | [node pool stat](#node-pool-stat) |  |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/nodeStat | [node stat](#node-stat) |  |
| GET | /test/screst/nodeStat | [node stat operation](#node-stat-operation) |  |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/openchallenges | [openchallenges](#openchallenges) |  |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d3/personalPeriodicLimit | [personal periodic limit](#personal-periodic-limit) |  |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d3/pourAmount | [pour amount](#pour-amount) |  |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/provider-rewards | [provider rewards](#provider-rewards) |  |
| POST | /v1/transaction/put | [put transaction](#put-transaction) | PutTransaction - Put a transaction to the transaction pool. Transaction size cannot exceed the max payload size which is a global configuration of the chain. |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/readmarkers | [readmarkers](#readmarkers) |  |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/replicate-authorizer-aggregate | [replicate authorizer aggregates](#replicate-authorizer-aggregates) | Gets list of authorizer aggregate records |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/replicate-blobber-aggregate | [replicate blobber aggregates](#replicate-blobber-aggregates) | Gets list of blobber aggregate records |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/replicate-miner-aggregate | [replicate miner aggregates](#replicate-miner-aggregates) | Gets list of miner aggregate records |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/replicate-sharder-aggregate | [replicate sharder aggregates](#replicate-sharder-aggregates) | Gets list of sharder aggregate records |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/replicate-snapshots | [replicate snapshots](#replicate-snapshots) | Gets list of global snapshot records |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/replicate-user-aggregate | [replicate user aggregates](#replicate-user-aggregates) | Gets list of user aggregate records |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/replicate-validator-aggregate | [replicate validator aggregates](#replicate-validator-aggregates) | Gets list of validator aggregate records |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/search | [search](#search) | Generic search endpoint. |
| GET | /v1/sharder/get/stats | [sharderstats](#sharderstats) |  |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/storage-config | [storage config](#storage-config) |  |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/transaction | [transaction](#transaction) |  |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/transactions | [transactions](#transactions) | Gets filtered list of transaction information. The list is filtered on the first valid input, or otherwise all the endpoint returns all translations. |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/validators | [validators](#validators) | Get a list of validators based on activity and stakability. Retrieves a list of validators, optionally filtered by whether they are active and/or stakable. |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/vesting_config | [vesting config](#vesting-config) |  |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/writemarkers | [writemarkers](#writemarkers) |  |
  


## Paths

### <span id="get-alloc-blobber-terms"></span> Get terms of storage service for a specific allocation and blobber (write_price, read_price) if blobber_id is specified, otherwise, get terms of service for all blobbers of the allocation. (*GetAllocBlobberTerms*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/alloc-blobber-term
```

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| allocation_id | `query` | string | `string` |  | ✓ |  | id of allocation |
| blobber_id | `query` | string | `string` |  |  |  | id of blobber |
| limit | `query` | string | `string` |  |  |  | limit |
| offset | `query` | string | `string` |  |  |  | offset |
| sort | `query` | string | `string` |  |  |  | desc or asc |

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

### <span id="alloc-blobbers"></span> returns list of all active blobbers that match the allocation request, or an error if not enough blobbers are available. (*alloc_blobbers*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/alloc_blobbers
```

Before the user attempts to create an allocation, they can use this endpoint to get a list of blobbers that match the allocation request. This includes:

Read and write price ranges
Data and parity shards
Size
Restricted status

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| allocation_data | `query` | string | `string` |  | ✓ |  | Allocation request data, in valid JSON format, following the allocationBlobbersRequest struct. |
| limit | `query` | string | `string` |  |  |  | limit |
| offset | `query` | string | `string` |  |  |  | offset |
| sort | `query` | string | `string` |  |  |  | desc or asc |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#alloc-blobbers-200) | OK | stringArray |  | [schema](#alloc-blobbers-200-schema) |
| [400](#alloc-blobbers-400) | Bad Request |  |  | [schema](#alloc-blobbers-400-schema) |

#### Responses


##### <span id="alloc-blobbers-200"></span> 200 - stringArray
Status: OK

###### <span id="alloc-blobbers-200-schema"></span> Schema
   
  


 [StringArray](#string-array)

##### <span id="alloc-blobbers-400"></span> 400
Status: Bad Request

###### <span id="alloc-blobbers-400-schema"></span> Schema

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

### <span id="allocation"></span> allocation (*allocation*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/allocation
```

Gets allocation object

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| allocation | `query` | string | `string` |  | ✓ |  | Id of the allocation to get |

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

### <span id="allocation-update-min-lock"></span> Calculates the cost for updating an allocation. (*allocation-update-min-lock*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/allocation-update-min-lock
```

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| data | `query` | string | `string` |  | ✓ |  | Update allocation request data, in valid JSON format, following the updateAllocationRequest struct. |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#allocation-update-min-lock-200) | OK | AllocationUpdateMinLockResponse |  | [schema](#allocation-update-min-lock-200-schema) |
| [400](#allocation-update-min-lock-400) | Bad Request |  |  | [schema](#allocation-update-min-lock-400-schema) |
| [500](#allocation-update-min-lock-500) | Internal Server Error |  |  | [schema](#allocation-update-min-lock-500-schema) |

#### Responses


##### <span id="allocation-update-min-lock-200"></span> 200 - AllocationUpdateMinLockResponse
Status: OK

###### <span id="allocation-update-min-lock-200-schema"></span> Schema
   
  

[AllocationUpdateMinLockResponse](#allocation-update-min-lock-response)

##### <span id="allocation-update-min-lock-400"></span> 400
Status: Bad Request

###### <span id="allocation-update-min-lock-400-schema"></span> Schema

##### <span id="allocation-update-min-lock-500"></span> 500
Status: Internal Server Error

###### <span id="allocation-update-min-lock-500-schema"></span> Schema

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

### <span id="blobber-allocations"></span> blobber allocations (*blobber-allocations*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/blobber-allocations
```

Gets a list of allocation information for allocations owned by the client

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| blobber_id | `query` | string | `string` |  | ✓ |  | blobber id of allocations we wish to list |
| limit | `query` | string | `string` |  |  |  | limit |
| offset | `query` | string | `string` |  |  |  | offset |
| sort | `query` | string | `string` |  |  |  | desc or asc by created date |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#blobber-allocations-200) | OK | StorageAllocation |  | [schema](#blobber-allocations-200-schema) |
| [400](#blobber-allocations-400) | Bad Request |  |  | [schema](#blobber-allocations-400-schema) |
| [500](#blobber-allocations-500) | Internal Server Error |  |  | [schema](#blobber-allocations-500-schema) |

#### Responses


##### <span id="blobber-allocations-200"></span> 200 - StorageAllocation
Status: OK

###### <span id="blobber-allocations-200-schema"></span> Schema
   
  

[][StorageAllocation](#storage-allocation)

##### <span id="blobber-allocations-400"></span> 400
Status: Bad Request

###### <span id="blobber-allocations-400-schema"></span> Schema

##### <span id="blobber-allocations-500"></span> 500
Status: Internal Server Error

###### <span id="blobber-allocations-500-schema"></span> Schema

### <span id="blobber-challenges"></span> blobber challenges (*blobber-challenges*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/blobber-challenges
```

Gets list of challenges for a blobber in a specific time interval by blobber id

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| from | `query` | string | `string` |  | ✓ |  | start time of the interval for which to get challenges (epoch timestamp in seconds) |
| id | `query` | string | `string` |  | ✓ |  | id of blobber for which to get challenges |
| to | `query` | string | `string` |  | ✓ |  | end time of interval for which to get challenges (epoch timestamp in seconds) |

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

### <span id="blobber-ids"></span> blobber ids (*blobber_ids*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/blobber_ids
```

convert list of blobber urls into ids

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| blobber_urls | `query` | string | `string` |  | ✓ |  | list of blobber URLs |
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

### <span id="block"></span> block (*block*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/block
```

Gets block information

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| block_hash | `query` | string | `string` |  |  |  | Hash (or identifier) of the block |
| date | `query` | string | `string` |  |  |  | block created closest to the date (epoch timestamp in seconds) |
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

### <span id="collected-reward"></span> Returns collected reward for a client_id. (*collected_reward*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/collected_reward
```

> Note: start-date and end-date resolves to the closest block number for those timestamps on the network.

> Note: Using start/end-block and start/end-date together would only return results with start/end-block

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| client-id | `query` | string | `string` |  | ✓ |  | ID of the client for which to get rewards |
| data-points | `query` | string | `string` |  |  |  | number of data points in response |
| end-block | `query` | string | `string` |  |  |  | end block number till which to collect rewards |
| end-date | `query` | string | `string` |  |  |  | end date till which to collect rewards |
| start-block | `query` | string | `string` |  |  |  | start block number from which to start collecting rewards |
| start-date | `query` | string | `string` |  |  |  | start date from which to start collecting rewards |

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
| [200](#delegate-rewards-200) | OK | RewardDelegate |  | [schema](#delegate-rewards-200-schema) |
| [400](#delegate-rewards-400) | Bad Request |  |  | [schema](#delegate-rewards-400-schema) |
| [500](#delegate-rewards-500) | Internal Server Error |  |  | [schema](#delegate-rewards-500-schema) |

#### Responses


##### <span id="delegate-rewards-200"></span> 200 - RewardDelegate
Status: OK

###### <span id="delegate-rewards-200-schema"></span> Schema
   
  

[][RewardDelegate](#reward-delegate)

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
| transaction_hash | `query` | string | `string` |  | ✓ |  | Hash of the transactions to get errors of. |

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

Before the user attempts to create a free allocation, they can use this endpoint to get a list of blobbers that match the allocation request. This includes:

Read and write price ranges
Data and parity shards
Size
Restricted status

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| free_allocation_data | `query` | string | `string` |  | ✓ |  | Free Allocation request data, in valid JSON format, following the freeStorageAllocationInput struct. |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#free-alloc-blobbers-200) | OK | stringArray |  | [schema](#free-alloc-blobbers-200-schema) |
| [400](#free-alloc-blobbers-400) | Bad Request |  |  | [schema](#free-alloc-blobbers-400-schema) |

#### Responses


##### <span id="free-alloc-blobbers-200"></span> 200 - stringArray
Status: OK

###### <span id="free-alloc-blobbers-200-schema"></span> Schema
   
  


 [StringArray](#string-array)

##### <span id="free-alloc-blobbers-400"></span> 400
Status: Bad Request

###### <span id="free-alloc-blobbers-400-schema"></span> Schema

### <span id="get-authorizer"></span> get authorizer (*getAuthorizer*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d3/getAuthorizer
```

get details of a given authorizer ID

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
| blobber_id | `query` | string | `string` |  | ✓ |  | blobber for which to return information from the sharders |

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

### <span id="get-expired-allocations"></span> Get expired allocations for a specific blobber. Retrieves a list of expired allocations associated with a specified blobber. (*getExpiredAllocations*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/getExpiredAllocations
```

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| blobber_id | `query` | string | `string` |  | ✓ |  | The identifier of the blobber to retrieve expired allocations for. |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#get-expired-allocations-200) | OK | StorageAllocation |  | [schema](#get-expired-allocations-200-schema) |
| [500](#get-expired-allocations-500) | Internal Server Error |  |  | [schema](#get-expired-allocations-500-schema) |

#### Responses


##### <span id="get-expired-allocations-200"></span> 200 - StorageAllocation
Status: OK

###### <span id="get-expired-allocations-200-schema"></span> Schema
   
  

[StorageAllocation](#storage-allocation)

##### <span id="get-expired-allocations-500"></span> 500
Status: Internal Server Error

###### <span id="get-expired-allocations-500-schema"></span> Schema

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
| provider_type | `query` | string | `string` |  | ✓ |  | type of the provider, possible values are 3 (blobber), 4 (validator), 5 (authorizer) |

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
| limit | `query` | string | `string` |  |  |  | Maximum number of results to return. |
| offset | `query` | string | `string` |  |  |  | Pagination offset to specify the starting point of the result set. |
| sort | `query` | string | `string` |  |  |  | desc or asc |

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

Gets writemarkers according to a filter

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| allocation_id | `query` | string | `string` |  | ✓ |  | List write markers for this allocation |
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

### <span id="get-blocks"></span> Gets block information for all blocks. Todo: We need to add a filter to this. (*get_blocks*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/get_blocks
```

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| block_hash | `query` | string | `string` |  | ✓ |  | block hash |
| end | `query` | string | `string` |  |  |  | Ending block number for the range of blocks to retrieve. |
| limit | `query` | string | `string` |  |  |  | limit |
| offset | `query` | string | `string` |  |  |  | offset |
| sort | `query` | string | `string` |  |  |  | desc or asc |
| start | `query` | string | `string` |  |  |  | Starting block number for the range of blocks to retrieve. |

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
| [200](#get-validator-200) | OK | validatorNodeResponse |  | [schema](#get-validator-200-schema) |
| [400](#get-validator-400) | Bad Request |  |  | [schema](#get-validator-400-schema) |
| [500](#get-validator-500) | Internal Server Error |  |  | [schema](#get-validator-500-schema) |

#### Responses


##### <span id="get-validator-200"></span> 200 - validatorNodeResponse
Status: OK

###### <span id="get-validator-200-schema"></span> Schema
   
  

[ValidatorNodeResponse](#validator-node-response)

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

### <span id="hardfork"></span> hardfork (*hardfork*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9/hardfork
```

get hardfork by name

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#hardfork-200) | OK | StringMap |  | [schema](#hardfork-200-schema) |
| [400](#hardfork-400) | Bad Request |  |  | [schema](#hardfork-400-schema) |
| [484](#hardfork-484) | Status 484 |  |  | [schema](#hardfork-484-schema) |

#### Responses


##### <span id="hardfork-200"></span> 200 - StringMap
Status: OK

###### <span id="hardfork-200-schema"></span> Schema
   
  

[StringMap](#string-map)

##### <span id="hardfork-400"></span> 400
Status: Bad Request

###### <span id="hardfork-400-schema"></span> Schema

##### <span id="hardfork-484"></span> 484
Status: Status 484

###### <span id="hardfork-484-schema"></span> Schema

### <span id="latestreadmarker"></span> latestreadmarker (*latestreadmarker*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/latestreadmarker
```

Gets latest read marker for a client and blobber

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| allocation | `query` | string | `string` |  |  |  | Allocation ID associated with the read marker. |
| blobber | `query` | string | `string` |  |  |  | blobber ID associated with the read marker. |
| client | `query` | string | `string` |  |  |  | ID of the client for which to get the latest read marker. |

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

lists node pool stats for a given client

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| id | `query` | string | `string` |  | ✓ |  | miner node ID |
| pool_id | `query` | string | `string` |  |  |  | pool_id |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#node-pool-stat-200) | OK | NodePool |  | [schema](#node-pool-stat-200-schema) |
| [400](#node-pool-stat-400) | Bad Request |  |  | [schema](#node-pool-stat-400-schema) |
| [484](#node-pool-stat-484) | Status 484 |  |  | [schema](#node-pool-stat-484-schema) |

#### Responses


##### <span id="node-pool-stat-200"></span> 200 - NodePool
Status: OK

###### <span id="node-pool-stat-200-schema"></span> Schema
   
  

[][NodePool](#node-pool)

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
| id | `query` | string | `string` |  | ✓ |  | miner or sharder ID |

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
| [484](#node-stat-operation-484) | Status 484 |  |  | [schema](#node-stat-operation-484-schema) |

#### Responses


##### <span id="node-stat-operation-200"></span> 200 - nodeStat
Status: OK

###### <span id="node-stat-operation-200-schema"></span> Schema
   
  

[NodeStat](#node-stat)

##### <span id="node-stat-operation-400"></span> 400
Status: Bad Request

###### <span id="node-stat-operation-400-schema"></span> Schema

##### <span id="node-stat-operation-484"></span> 484
Status: Status 484

###### <span id="node-stat-operation-484-schema"></span> Schema

### <span id="openchallenges"></span> openchallenges (*openchallenges*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/openchallenges
```

Gets open challenges for a blobber

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| blobber | `query` | string | `string` |  | ✓ |  | id of blobber for which to get open challenges |
| from | `query` | string | `string` |  |  |  | Starting round number for fetching challenges. |
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

list minersc config settings for given client_id

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

returns the value of smart_contracts.faucetsc.pour_amount configured in sc.yaml

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#pour-amount-200) | OK | MinerSCPourAmount |  | [schema](#pour-amount-200-schema) |
| [404](#pour-amount-404) | Not Found |  |  | [schema](#pour-amount-404-schema) |

#### Responses


##### <span id="pour-amount-200"></span> 200 - MinerSCPourAmount
Status: OK

###### <span id="pour-amount-200-schema"></span> Schema
   
  

[MinerSCPourAmount](#miner-s-c-pour-amount)

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
| [200](#provider-rewards-200) | OK | RewardProvider |  | [schema](#provider-rewards-200-schema) |
| [400](#provider-rewards-400) | Bad Request |  |  | [schema](#provider-rewards-400-schema) |
| [500](#provider-rewards-500) | Internal Server Error |  |  | [schema](#provider-rewards-500-schema) |

#### Responses


##### <span id="provider-rewards-200"></span> 200 - RewardProvider
Status: OK

###### <span id="provider-rewards-200-schema"></span> Schema
   
  

[][RewardProvider](#reward-provider)

##### <span id="provider-rewards-400"></span> 400
Status: Bad Request

###### <span id="provider-rewards-400-schema"></span> Schema

##### <span id="provider-rewards-500"></span> 500
Status: Internal Server Error

###### <span id="provider-rewards-500-schema"></span> Schema

### <span id="put-transaction"></span> PutTransaction - Put a transaction to the transaction pool. Transaction size cannot exceed the max payload size which is a global configuration of the chain. (*putTransaction*)

```
POST /v1/transaction/put
```

#### Consumes
  * application/json

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| Transaction | `body` | integer | `int64` | | ✓ | |  |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#put-transaction-200) | OK |  |  | [schema](#put-transaction-200-schema) |
| [400](#put-transaction-400) | Bad Request |  |  | [schema](#put-transaction-400-schema) |
| [500](#put-transaction-500) | Internal Server Error |  |  | [schema](#put-transaction-500-schema) |

#### Responses


##### <span id="put-transaction-200"></span> 200
Status: OK

###### <span id="put-transaction-200-schema"></span> Schema

##### <span id="put-transaction-400"></span> 400
Status: Bad Request

###### <span id="put-transaction-400-schema"></span> Schema

##### <span id="put-transaction-500"></span> 500
Status: Internal Server Error

###### <span id="put-transaction-500-schema"></span> Schema

### <span id="readmarkers"></span> readmarkers (*readmarkers*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/readmarkers
```

Gets read markers according to a filter

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| allocation_id | `query` | string | `string` |  |  |  | filter in only read markers by this allocation |
| auth_ticket | `query` | string | `string` |  |  |  | filter in only read markers using this auth ticket |
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

### <span id="replicate-authorizer-aggregates"></span> Gets list of authorizer aggregate records (*replicateAuthorizerAggregates*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/replicate-authorizer-aggregate
```

> Note: This endpoint is DEPRECATED and will be removed in the next release.

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| limit | `query` | string | `string` |  |  |  | limit |
| offset | `query` | string | `string` |  |  |  | offset |
| round | `query` | string | `string` |  |  |  | round number to start from |
| sort | `query` | string | `string` |  |  |  | desc or asc |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#replicate-authorizer-aggregates-200) | OK | AuthorizerAggregate |  | [schema](#replicate-authorizer-aggregates-200-schema) |
| [500](#replicate-authorizer-aggregates-500) | Internal Server Error |  |  | [schema](#replicate-authorizer-aggregates-500-schema) |

#### Responses


##### <span id="replicate-authorizer-aggregates-200"></span> 200 - AuthorizerAggregate
Status: OK

###### <span id="replicate-authorizer-aggregates-200-schema"></span> Schema
   
  

[AuthorizerAggregate](#authorizer-aggregate)

##### <span id="replicate-authorizer-aggregates-500"></span> 500
Status: Internal Server Error

###### <span id="replicate-authorizer-aggregates-500-schema"></span> Schema

### <span id="replicate-blobber-aggregates"></span> Gets list of blobber aggregate records (*replicateBlobberAggregates*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/replicate-blobber-aggregate
```

> Note: This endpoint is DEPRECATED and will be removed in the next release.

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| limit | `query` | string | `string` |  |  |  | limit |
| offset | `query` | string | `string` |  |  |  | offset |
| round | `query` | string | `string` |  |  |  | round number to start from |
| sort | `query` | string | `string` |  |  |  | desc or asc |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#replicate-blobber-aggregates-200) | OK | BlobberAggregate |  | [schema](#replicate-blobber-aggregates-200-schema) |
| [500](#replicate-blobber-aggregates-500) | Internal Server Error |  |  | [schema](#replicate-blobber-aggregates-500-schema) |

#### Responses


##### <span id="replicate-blobber-aggregates-200"></span> 200 - BlobberAggregate
Status: OK

###### <span id="replicate-blobber-aggregates-200-schema"></span> Schema
   
  

[BlobberAggregate](#blobber-aggregate)

##### <span id="replicate-blobber-aggregates-500"></span> 500
Status: Internal Server Error

###### <span id="replicate-blobber-aggregates-500-schema"></span> Schema

### <span id="replicate-miner-aggregates"></span> Gets list of miner aggregate records (*replicateMinerAggregates*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/replicate-miner-aggregate
```

> Note: This endpoint is DEPRECATED and will be removed in the next release.

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| limit | `query` | string | `string` |  |  |  | limit |
| offset | `query` | string | `string` |  |  |  | offset |
| round | `query` | string | `string` |  |  |  | round number to start from |
| sort | `query` | string | `string` |  |  |  | desc or asc |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#replicate-miner-aggregates-200) | OK | MinerAggregate |  | [schema](#replicate-miner-aggregates-200-schema) |
| [500](#replicate-miner-aggregates-500) | Internal Server Error |  |  | [schema](#replicate-miner-aggregates-500-schema) |

#### Responses


##### <span id="replicate-miner-aggregates-200"></span> 200 - MinerAggregate
Status: OK

###### <span id="replicate-miner-aggregates-200-schema"></span> Schema
   
  

[MinerAggregate](#miner-aggregate)

##### <span id="replicate-miner-aggregates-500"></span> 500
Status: Internal Server Error

###### <span id="replicate-miner-aggregates-500-schema"></span> Schema

### <span id="replicate-sharder-aggregates"></span> Gets list of sharder aggregate records (*replicateSharderAggregates*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/replicate-sharder-aggregate
```

> Note: This endpoint is DEPRECATED and will be removed in the next release.

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| limit | `query` | string | `string` |  |  |  | limit |
| offset | `query` | string | `string` |  |  |  | offset |
| round | `query` | string | `string` |  |  |  | round number to start from |
| sort | `query` | string | `string` |  |  |  | desc or asc |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#replicate-sharder-aggregates-200) | OK | SharderAggregate |  | [schema](#replicate-sharder-aggregates-200-schema) |
| [500](#replicate-sharder-aggregates-500) | Internal Server Error |  |  | [schema](#replicate-sharder-aggregates-500-schema) |

#### Responses


##### <span id="replicate-sharder-aggregates-200"></span> 200 - SharderAggregate
Status: OK

###### <span id="replicate-sharder-aggregates-200-schema"></span> Schema
   
  

[SharderAggregate](#sharder-aggregate)

##### <span id="replicate-sharder-aggregates-500"></span> 500
Status: Internal Server Error

###### <span id="replicate-sharder-aggregates-500-schema"></span> Schema

### <span id="replicate-snapshots"></span> Gets list of global snapshot records (*replicateSnapshots*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/replicate-snapshots
```

> Note: This endpoint is DEPRECATED and will be removed in the next release.

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| limit | `query` | string | `string` |  |  |  | limit |
| offset | `query` | string | `string` |  |  |  | offset |
| round | `query` | string | `string` |  |  |  | round number to start from |
| sort | `query` | string | `string` |  |  |  | desc or asc |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#replicate-snapshots-200) | OK | Snapshot |  | [schema](#replicate-snapshots-200-schema) |
| [500](#replicate-snapshots-500) | Internal Server Error |  |  | [schema](#replicate-snapshots-500-schema) |

#### Responses


##### <span id="replicate-snapshots-200"></span> 200 - Snapshot
Status: OK

###### <span id="replicate-snapshots-200-schema"></span> Schema
   
  

[][Snapshot](#snapshot)

##### <span id="replicate-snapshots-500"></span> 500
Status: Internal Server Error

###### <span id="replicate-snapshots-500-schema"></span> Schema

### <span id="replicate-user-aggregates"></span> Gets list of user aggregate records (*replicateUserAggregates*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/replicate-user-aggregate
```

> Note: This endpoint is DEPRECATED and will be removed in the next release.

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| limit | `query` | string | `string` |  |  |  | limit |
| offset | `query` | string | `string` |  |  |  | offset |
| round | `query` | string | `string` |  |  |  | round number to start from |
| sort | `query` | string | `string` |  |  |  | desc or asc |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#replicate-user-aggregates-200) | OK | UserAggregate |  | [schema](#replicate-user-aggregates-200-schema) |
| [500](#replicate-user-aggregates-500) | Internal Server Error |  |  | [schema](#replicate-user-aggregates-500-schema) |

#### Responses


##### <span id="replicate-user-aggregates-200"></span> 200 - UserAggregate
Status: OK

###### <span id="replicate-user-aggregates-200-schema"></span> Schema
   
  

[UserAggregate](#user-aggregate)

##### <span id="replicate-user-aggregates-500"></span> 500
Status: Internal Server Error

###### <span id="replicate-user-aggregates-500-schema"></span> Schema

### <span id="replicate-validator-aggregates"></span> Gets list of validator aggregate records (*replicateValidatorAggregates*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/replicate-validator-aggregate
```

> Note: This endpoint is DEPRECATED and will be removed in the next release.

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| limit | `query` | string | `string` |  |  |  | limit |
| offset | `query` | string | `string` |  |  |  | offset |
| round | `query` | string | `string` |  |  |  | round number to start from |
| sort | `query` | string | `string` |  |  |  | desc or asc |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#replicate-validator-aggregates-200) | OK | ValidatorAggregate |  | [schema](#replicate-validator-aggregates-200-schema) |
| [500](#replicate-validator-aggregates-500) | Internal Server Error |  |  | [schema](#replicate-validator-aggregates-500-schema) |

#### Responses


##### <span id="replicate-validator-aggregates-200"></span> 200 - ValidatorAggregate
Status: OK

###### <span id="replicate-validator-aggregates-200-schema"></span> Schema
   
  

[ValidatorAggregate](#validator-aggregate)

##### <span id="replicate-validator-aggregates-500"></span> 500
Status: Internal Server Error

###### <span id="replicate-validator-aggregates-500-schema"></span> Schema

### <span id="search"></span> Generic search endpoint. (*search*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/search
```

Integer If the input can be converted to an integer, it is interpreted as a round number and information for the
matching block is returned. Otherwise, the input is treated as string and matched against block hash,
transaction hash, user id.
If a match is found the matching object is returned.

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| searchString | `query` | string | `string` |  | ✓ |  | Generic query string, supported inputs: Block hash, Round num, Transaction hash, Wallet address |

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
| [200](#sharderstats-200) | OK | ExplorerStats |  | [schema](#sharderstats-200-schema) |
| [404](#sharderstats-404) | Not Found |  |  | [schema](#sharderstats-404-schema) |

#### Responses


##### <span id="sharderstats-200"></span> 200 - ExplorerStats
Status: OK

###### <span id="sharderstats-200-schema"></span> Schema
   
  

[ExplorerStats](#explorer-stats)

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

### <span id="transaction"></span> transaction (*transaction*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/transaction
```

Gets transaction information from transaction hash

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| transaction_hash | `query` | string | `string` |  | ✓ |  | The hash of the transaction to retrieve. |

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

### <span id="transactions"></span> Gets filtered list of transaction information. The list is filtered on the first valid input, or otherwise all the endpoint returns all translations. (*transactions*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/transactions
```

Filters processed in the order: client id, to client id, block hash and start, end blocks.

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| block_hash | `query` | string | `string` |  |  |  | restrict to transactions in indicated block |
| client_id | `query` | string | `string` |  |  |  | restrict to transactions sent by the specified client |
| end | `query` | string | `string` |  |  |  | restrict to transactions within specified start block and end block |
| limit | `query` | string | `string` |  |  |  | limit |
| offset | `query` | string | `string` |  |  |  | offset |
| sort | `query` | string | `string` |  |  |  | desc or asc |
| start | `query` | string | `string` |  |  |  | restrict to transactions within specified start block and end block |
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

### <span id="validators"></span> Get a list of validators based on activity and stakability. Retrieves a list of validators, optionally filtered by whether they are active and/or stakable. (*validators*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/validators
```

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| active | `query` | string | `string` |  |  |  | Filter validators based on whether they are currently active. Set to 'true' to filter only active validators. |
| limit | `query` | integer | `int64` |  |  |  | The maximum number of validators to return. |
| offset | `query` | integer | `int64` |  |  |  | The starting point for pagination. |
| order | `query` | string | `string` |  |  |  | Order of the validators returned, e.g., 'asc' for ascending. |
| stakable | `query` | string | `string` |  |  |  | Filter validators based on whether they are currently stakable. Set to 'true' to filter only stakable validators. |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#validators-200) | OK | validatorNodeResponse |  | [schema](#validators-200-schema) |
| [400](#validators-400) | Bad Request |  |  | [schema](#validators-400-schema) |

#### Responses


##### <span id="validators-200"></span> 200 - validatorNodeResponse
Status: OK

###### <span id="validators-200-schema"></span> Schema
   
  

[][ValidatorNodeResponse](#validator-node-response)

##### <span id="validators-400"></span> 400
Status: Bad Request

###### <span id="validators-400-schema"></span> Schema

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
| limit | `query` | string | `string` |  |  |  | limit |
| offset | `query` | string | `string` |  |  |  | offset |
| sort | `query` | string | `string` |  |  |  | asc or desc |

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



### <span id="allocation-update-min-lock-response"></span> AllocationUpdateMinLockResponse


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| MinLockDemand | int64 (formatted integer)| `int64` |  | |  |  |



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



### <span id="blobber-allocation"></span> BlobberAllocation


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| AllocationID | string| `string` |  | |  |  |
| AllocationRoot | string| `string` |  | |  |  |
| BlobberID | string| `string` |  | |  |  |
| Size | int64 (formatted integer)| `int64` |  | | Size is blobber allocation maximum size |  |
| challenge_pool_integral_value | [Coin](#coin)| `Coin` |  | |  |  |
| challenge_reward | [Coin](#coin)| `Coin` |  | |  |  |
| latest_finalized_chall_created_att | [Timestamp](#timestamp)| `Timestamp` |  | |  |  |
| latest_successful_chall_created_at | [Timestamp](#timestamp)| `Timestamp` |  | |  |  |
| penalty | [Coin](#coin)| `Coin` |  | |  |  |
| read_reward | [Coin](#coin)| `Coin` |  | |  |  |
| returned | [Coin](#coin)| `Coin` |  | |  |  |
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



### <span id="chain-stats"></span> ChainStats


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| Count | int64 (formatted integer)| `int64` |  | |  |  |
| CurrentRound | int64 (formatted integer)| `int64` |  | |  |  |
| LastFinalizedRound | int64 (formatted integer)| `int64` |  | |  |  |
| Max | double (formatted number)| `float64` |  | |  |  |
| Mean | double (formatted number)| `float64` |  | |  |  |
| Min | double (formatted number)| `float64` |  | |  |  |
| Percentile50 | double (formatted number)| `float64` |  | |  |  |
| Percentile90 | double (formatted number)| `float64` |  | |  |  |
| Percentile95 | double (formatted number)| `float64` |  | |  |  |
| Percentile99 | double (formatted number)| `float64` |  | |  |  |
| Rate1 | double (formatted number)| `float64` |  | |  |  |
| Rate15 | double (formatted number)| `float64` |  | |  |  |
| Rate5 | double (formatted number)| `float64` |  | |  |  |
| RateMean | double (formatted number)| `float64` |  | |  |  |
| RunningTxnCount | int64 (formatted integer)| `int64` |  | |  |  |
| StdDev | double (formatted number)| `float64` |  | |  |  |
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

### <span id="challenges-response"></span> ChallengesResponse


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| BlobberID | string| `string` |  | |  |  |
| Challenges | [][StorageChallengeResponse](#storage-challenge-response)| `[]*StorageChallengeResponse` |  | |  |  |



### <span id="client"></span> Client


> go:generate msgp -io=false -tests=false -v
  





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



### <span id="explorer-stats"></span> ExplorerStats


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| AverageBlockSize | int64 (formatted integer)| `int64` |  | |  |  |
| LastFinalizedRound | int64 (formatted integer)| `int64` |  | |  |  |
| MeanScanBlockStatsTime | double (formatted number)| `float64` |  | |  |  |
| PrevInvocationCount | uint64 (formatted integer)| `uint64` |  | |  |  |
| PrevInvocationScanTime | string| `string` |  | |  |  |
| StateHealth | int64 (formatted integer)| `int64` |  | |  |  |



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



### <span id="miner-dto-node"></span> MinerDtoNode


  



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



### <span id="miner-s-c-pour-amount"></span> MinerSCPourAmount


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| pour_amount | [Coin](#coin)| `Coin` |  | |  |  |



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



### <span id="reward-round"></span> RewardRound


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| StartRound | int64 (formatted integer)| `int64` |  | |  |  |
| timestamp | [Timestamp](#timestamp)| `Timestamp` |  | |  |  |



### <span id="settings"></span> Settings


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| DelegateWallet | string| `string` |  | |  |  |
| MaxNumDelegates | int64 (formatted integer)| `int64` |  | |  |  |
| ServiceChargeRatio | double (formatted number)| `float64` |  | |  |  |



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



### <span id="simple-dto-node"></span> SimpleDtoNode


  



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



### <span id="storage-allocation"></span> StorageAllocation


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| BlobberAllocs | [][BlobberAllocation](#blobber-allocation)| `[]*BlobberAllocation` |  | | Blobbers not to be used anywhere except /allocation and /allocations table</br>if Blobbers are getting used in any smart-contract, we should avoid. |  |
| Canceled | boolean| `bool` |  | | Canceled set to true where allocation finalized by cancel_allocation</br>transaction. |  |
| DataShards | int64 (formatted integer)| `int64` |  | |  |  |
| DiverseBlobbers | boolean| `bool` |  | |  |  |
| FileOptions | uint16 (formatted integer)| `uint16` |  | | FileOptions to define file restrictions on an allocation for third-parties</br>default 00000000 for all crud operations suggesting only owner has the below listed abilities.</br>enabling option/s allows any third party to perform certain ops</br>00000001 - 1  - upload</br>00000010 - 2  - delete</br>00000100 - 4  - update</br>00001000 - 8  - move</br>00010000 - 16 - copy</br>00100000 - 32 - rename |  |
| Finalized | boolean| `bool` |  | | Finalized is true where allocation has been finalized. |  |
| ID | string| `string` |  | | ID is unique allocation ID that is equal to hash of transaction with</br>which the allocation has created. |  |
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
| Responded | int64 (formatted integer)| `int64` |  | |  |  |
| RoundCreatedAt | int64 (formatted integer)| `int64` |  | |  |  |
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
| Responded | int64 (formatted integer)| `int64` |  | |  |  |
| RoundCreatedAt | int64 (formatted integer)| `int64` |  | |  |  |
| Seed | int64 (formatted integer)| `int64` |  | |  |  |
| TotalValidators | int64 (formatted integer)| `int64` |  | |  |  |
| ValidatorIDs | []string| `[]string` |  | |  |  |
| Validators | [][ValidationNode](#validation-node)| `[]*ValidationNode` |  | |  |  |
| created | [Timestamp](#timestamp)| `Timestamp` |  | |  |  |
| timestamp | [Timestamp](#timestamp)| `Timestamp` |  | |  |  |



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
| read_price | [Coin](#coin)| `Coin` |  | |  |  |
| write_price | [Coin](#coin)| `Coin` |  | |  |  |



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
| BlockHash | string| `string` |  | |  |  |
| ClientId | string| `string` |  | |  |  |
| CreatedAt | date-time (formatted string)| `strfmt.DateTime` |  | |  |  |
| CreationDate | int64 (formatted integer)| `int64` |  | |  |  |
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
| Version | string| `string` |  | |  |  |
| fee | [Coin](#coin)| `Coin` |  | |  |  |
| value | [Coin](#coin)| `Coin` |  | |  |  |



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



### <span id="validation-node"></span> ValidationNode


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| BaseURL | string| `string` |  | |  |  |
| HasBeenKilled | boolean| `bool` |  | |  |  |
| HasBeenShutDown | boolean| `bool` |  | |  |  |
| ID | string| `string` |  | |  |  |
| last_health_check | [Timestamp](#timestamp)| `Timestamp` |  | |  |  |
| provider_type | [Provider](#provider)| `Provider` |  | |  |  |
| stake_pool_settings | [Settings](#settings)| `Settings` |  | |  |  |



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



### <span id="version-field"></span> VersionField


> go:generate msgp -io=false -tests=false -v
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| Version | string| `string` |  | |  |  |



### <span id="wrapper"></span> Wrapper


  

[interface{}](#interface)

### <span id="write-marker"></span> WriteMarker


  

[interface{}](#interface)

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
| NumDelegates | int64 (formatted integer)| `int64` |  | |  |  |
| ServiceCharge | double (formatted number)| `float64` |  | |  |  |
| URL | string| `string` |  | |  |  |
| fee | [Coin](#coin)| `Coin` |  | |  |  |



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



### <span id="free-storage-allocation-input"></span> freeStorageAllocationInput


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| Blobbers | []string| `[]string` |  | |  |  |
| Marker | string| `string` |  | |  |  |
| RecipientPublicKey | string| `string` |  | |  |  |



### <span id="full-block"></span> fullBlock


  



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



### <span id="storage-node-response"></span> storageNodeResponse


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| Allocated | int64 (formatted integer)| `int64` |  | |  |  |
| BaseURL | string| `string` |  | |  |  |
| Capacity | int64 (formatted integer)| `int64` |  | |  |  |
| ChallengesCompleted | int64 (formatted integer)| `int64` |  | |  |  |
| ChallengesPassed | int64 (formatted integer)| `int64` |  | |  |  |
| CreatedAt | date-time (formatted string)| `strfmt.DateTime` |  | |  |  |
| CreationRound | int64 (formatted integer)| `int64` |  | |  |  |
| DataReadLastRewardRound | double (formatted number)| `float64` |  | |  |  |
| ID | string| `string` |  | |  |  |
| IsKilled | boolean| `bool` |  | |  |  |
| IsRestricted | boolean| `bool` |  | |  |  |
| IsShutdown | boolean| `bool` |  | |  |  |
| LastRewardDataReadRound | int64 (formatted integer)| `int64` |  | |  |  |
| NotAvailable | boolean| `bool` |  | |  |  |
| ReadData | int64 (formatted integer)| `int64` |  | |  |  |
| SavedData | int64 (formatted integer)| `int64` |  | |  |  |
| StakedCapacity | int64 (formatted integer)| `int64` |  | |  |  |
| UsedAllocation | int64 (formatted integer)| `int64` |  | |  |  |
| last_health_check | [Timestamp](#timestamp)| `Timestamp` |  | |  |  |
| reward_round | [RewardRound](#reward-round)| `RewardRound` |  | |  |  |
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

### <span id="validator-node-response"></span> validatorNodeResponse


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| BaseUrl | string| `string` |  | |  |  |
| DelegateWallet | string| `string` |  | | StakePoolSettings |  |
| IsKilled | boolean| `bool` |  | |  |  |
| IsShutdown | boolean| `bool` |  | |  |  |
| NumDelegates | int64 (formatted integer)| `int64` |  | |  |  |
| PublicKey | string| `string` |  | |  |  |
| ServiceCharge | double (formatted number)| `float64` |  | |  |  |
| ValidatorID | string| `string` |  | |  |  |
| last_health_check | [Timestamp](#timestamp)| `Timestamp` |  | |  |  |
| stake_total | [Coin](#coin)| `Coin` |  | |  |  |
| total_service_charge | [Coin](#coin)| `Coin` |  | |  |  |
| uncollected_service_charge | [Coin](#coin)| `Coin` |  | |  |  |


