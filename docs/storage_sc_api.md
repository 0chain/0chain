


# Storage Smart Contract Public API:
  

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

###  operations

| Method  | URI     | Name   | Summary |
|---------|---------|--------|---------|
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/alloc-blobber-term | [get alloc blobber terms](#get-alloc-blobber-terms) | Get allocation/blobber terms of service. |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/alloc_blobbers | [get alloc blobbers](#get-alloc-blobbers) | Get blobbers for allocation request. |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/alloc_write_marker_count | [get alloc write marker count](#get-alloc-write-marker-count) | Count of write markers for an allocation. |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/allocation | [get allocation](#get-allocation) | Get allocation information |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/allocation-update-min-lock | [get allocation update min lock](#get-allocation-update-min-lock) | Calculates the cost for updating an allocation. |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/getWriteMarkers | [get allocation write markers](#get-allocation-write-markers) | Get write markers. |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/allocations | [get allocations](#get-allocations) | Get client allocations. |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/getBlobber | [get blobber](#get-blobber) | Get blobber information. |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/blobber-allocations | [get blobber allocations](#get-blobber-allocations) | Get blobber allocations. |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/blobber-challenges | [get blobber challenges](#get-blobber-challenges) | Get blobber challenges. |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/blobber_ids | [get blobber ids](#get-blobber-ids) | Get blobber ids by blobber urls. |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/getblobbers | [get blobbers](#get-blobbers) | Get active blobbers ids. |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/block | [get block](#get-block) | Gets block information |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/get_blocks | [get blocks](#get-blocks) | Get blocks for round range. |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/getchallenge | [get challenge](#get-challenge) | Get challenge information. |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/getChallengePoolStat | [get challenge pool stat](#get-challenge-pool-stat) | Get challenge pool statistics. |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/collected_reward | [get collected reward](#get-collected-reward) | Get collected reward. |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/getExpiredAllocations | [get expired allocations](#get-expired-allocations) | Get expired allocations. |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/free_alloc_blobbers | [get free alloc blobbers](#get-free-alloc-blobbers) | Get free allocation blobbers. |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/latestreadmarker | [get latest readmarker](#get-latest-readmarker) | Get latest read marker. |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/openchallenges | [get open challenges](#get-open-challenges) | Get blobber open challenges. |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/readmarkers | [get read markers](#get-read-markers) | Get read markers. |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/count_readmarkers | [get read markers count](#get-read-markers-count) | Gets read markers count. |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/getReadPoolStat | [get read pool stat](#get-read-pool-stat) | Get read pool statistics. |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/getStakePoolStat | [get stake pool stat](#get-stake-pool-stat) | Get stake pool statistics. |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/storage-config | [get storage config](#get-storage-config) | Get storage smart contract settings. |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/transaction | [get transaction](#get-transaction) | Get transaction information |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/errors | [get transaction errors](#get-transaction-errors) | Get transaction errors. |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/transactions | [get transactions](#get-transactions) | Get Transactions	list. |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/getUserStakePoolStat | [get user stake pool stat](#get-user-stake-pool-stat) | Get user stake pool statistics. |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/get_validator | [get validator](#get-validator) | Get validator information. |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/validators | [get validators](#get-validators) | Get validators. |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/writemarkers | [get write markers](#get-write-markers) | Get write markers. |
| POST | /v1/transaction/put | [put transaction](#put-transaction) | PutTransaction - Put a transaction to the transaction pool. Transaction size cannot exceed the max payload size which is a global configuration of the chain. |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/replicate-authorizer-aggregate | [replicate authorizer aggregates](#replicate-authorizer-aggregates) | Gets list of authorizer aggregate records |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/replicate-blobber-aggregate | [replicate blobber aggregates](#replicate-blobber-aggregates) | Gets list of blobber aggregate records |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/replicate-miner-aggregate | [replicate miner aggregates](#replicate-miner-aggregates) | Gets list of miner aggregate records |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/replicate-sharder-aggregate | [replicate sharder aggregates](#replicate-sharder-aggregates) | Gets list of sharder aggregate records |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/replicate-snapshots | [replicate snapshots](#replicate-snapshots) | Gets list of global snapshot records |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/replicate-user-aggregate | [replicate user aggregates](#replicate-user-aggregates) | Gets list of user aggregate records |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/replicate-validator-aggregate | [replicate validator aggregates](#replicate-validator-aggregates) | Gets list of validator aggregate records |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/search | [search](#search) | Generic search endpoint. |
  


## Paths

### <span id="get-alloc-blobber-terms"></span> Get allocation/blobber terms of service. (*GetAllocBlobberTerms*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/alloc-blobber-term
```

Get terms of storage service for a specific allocation and blobber (write_price, read_price) if blobber_id is specified.
Otherwise, get terms of service for all blobbers of the allocation.

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

### <span id="get-alloc-blobbers"></span> Get blobbers for allocation request. (*GetAllocBlobbers*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/alloc_blobbers
```

Returns list of all active blobbers that match the allocation request, or an error if not enough blobbers are available.
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
| [200](#get-alloc-blobbers-200) | OK | stringArray |  | [schema](#get-alloc-blobbers-200-schema) |
| [400](#get-alloc-blobbers-400) | Bad Request |  |  | [schema](#get-alloc-blobbers-400-schema) |

#### Responses


##### <span id="get-alloc-blobbers-200"></span> 200 - stringArray
Status: OK

###### <span id="get-alloc-blobbers-200-schema"></span> Schema
   
  


 [StringArray](#string-array)

##### <span id="get-alloc-blobbers-400"></span> 400
Status: Bad Request

###### <span id="get-alloc-blobbers-400-schema"></span> Schema

### <span id="get-alloc-write-marker-count"></span> Count of write markers for an allocation. (*GetAllocWriteMarkerCount*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/alloc_write_marker_count
```

Returns the count of write markers for an allocation given its id.

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| allocation_id | `query` | string | `string` |  | ✓ |  | allocation for which to get challenge pools statistics |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#get-alloc-write-marker-count-200) | OK | challengePoolStat |  | [schema](#get-alloc-write-marker-count-200-schema) |
| [400](#get-alloc-write-marker-count-400) | Bad Request |  |  | [schema](#get-alloc-write-marker-count-400-schema) |

#### Responses


##### <span id="get-alloc-write-marker-count-200"></span> 200 - challengePoolStat
Status: OK

###### <span id="get-alloc-write-marker-count-200-schema"></span> Schema
   
  

[ChallengePoolStat](#challenge-pool-stat)

##### <span id="get-alloc-write-marker-count-400"></span> 400
Status: Bad Request

###### <span id="get-alloc-write-marker-count-400-schema"></span> Schema

### <span id="get-allocation"></span> Get allocation information (*GetAllocation*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/allocation
```

Retrieves information about a specific allocation given its id.

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| allocation | `query` | string | `string` |  | ✓ |  | Id of the allocation to get |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#get-allocation-200) | OK | StorageAllocation |  | [schema](#get-allocation-200-schema) |
| [400](#get-allocation-400) | Bad Request |  |  | [schema](#get-allocation-400-schema) |
| [500](#get-allocation-500) | Internal Server Error |  |  | [schema](#get-allocation-500-schema) |

#### Responses


##### <span id="get-allocation-200"></span> 200 - StorageAllocation
Status: OK

###### <span id="get-allocation-200-schema"></span> Schema
   
  

[StorageAllocation](#storage-allocation)

##### <span id="get-allocation-400"></span> 400
Status: Bad Request

###### <span id="get-allocation-400-schema"></span> Schema

##### <span id="get-allocation-500"></span> 500
Status: Internal Server Error

###### <span id="get-allocation-500-schema"></span> Schema

### <span id="get-allocation-update-min-lock"></span> Calculates the cost for updating an allocation. (*GetAllocationUpdateMinLock*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/allocation-update-min-lock
```

Based on the allocation request data, this endpoint calculates the minimum lock demand for updating an allocation, which represents the cost of the allocation.

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| data | `query` | string | `string` |  | ✓ |  | Update allocation request data, in valid JSON format, following the updateAllocationRequest struct. |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#get-allocation-update-min-lock-200) | OK | AllocationUpdateMinLockResponse |  | [schema](#get-allocation-update-min-lock-200-schema) |
| [400](#get-allocation-update-min-lock-400) | Bad Request |  |  | [schema](#get-allocation-update-min-lock-400-schema) |
| [500](#get-allocation-update-min-lock-500) | Internal Server Error |  |  | [schema](#get-allocation-update-min-lock-500-schema) |

#### Responses


##### <span id="get-allocation-update-min-lock-200"></span> 200 - AllocationUpdateMinLockResponse
Status: OK

###### <span id="get-allocation-update-min-lock-200-schema"></span> Schema
   
  

[AllocationUpdateMinLockResponse](#allocation-update-min-lock-response)

##### <span id="get-allocation-update-min-lock-400"></span> 400
Status: Bad Request

###### <span id="get-allocation-update-min-lock-400-schema"></span> Schema

##### <span id="get-allocation-update-min-lock-500"></span> 500
Status: Internal Server Error

###### <span id="get-allocation-update-min-lock-500-schema"></span> Schema

### <span id="get-allocation-write-markers"></span> Get write markers. (*GetAllocationWriteMarkers*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/getWriteMarkers
```

Retrieves writemarkers of an allocation given the allocation id. Supports pagination.

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
| [200](#get-allocation-write-markers-200) | OK | WriteMarker |  | [schema](#get-allocation-write-markers-200-schema) |
| [400](#get-allocation-write-markers-400) | Bad Request |  |  | [schema](#get-allocation-write-markers-400-schema) |
| [500](#get-allocation-write-markers-500) | Internal Server Error |  |  | [schema](#get-allocation-write-markers-500-schema) |

#### Responses


##### <span id="get-allocation-write-markers-200"></span> 200 - WriteMarker
Status: OK

###### <span id="get-allocation-write-markers-200-schema"></span> Schema
   
  

[][WriteMarker](#write-marker)

##### <span id="get-allocation-write-markers-400"></span> 400
Status: Bad Request

###### <span id="get-allocation-write-markers-400-schema"></span> Schema

##### <span id="get-allocation-write-markers-500"></span> 500
Status: Internal Server Error

###### <span id="get-allocation-write-markers-500-schema"></span> Schema

### <span id="get-allocations"></span> Get client allocations. (*GetAllocations*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/allocations
```

Gets a list of allocation information for allocations owned by the client. Supports pagination.

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
| [200](#get-allocations-200) | OK | StorageAllocation |  | [schema](#get-allocations-200-schema) |
| [400](#get-allocations-400) | Bad Request |  |  | [schema](#get-allocations-400-schema) |
| [500](#get-allocations-500) | Internal Server Error |  |  | [schema](#get-allocations-500-schema) |

#### Responses


##### <span id="get-allocations-200"></span> 200 - StorageAllocation
Status: OK

###### <span id="get-allocations-200-schema"></span> Schema
   
  

[][StorageAllocation](#storage-allocation)

##### <span id="get-allocations-400"></span> 400
Status: Bad Request

###### <span id="get-allocations-400-schema"></span> Schema

##### <span id="get-allocations-500"></span> 500
Status: Internal Server Error

###### <span id="get-allocations-500-schema"></span> Schema

### <span id="get-blobber"></span> Get blobber information. (*GetBlobber*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/getBlobber
```

Retrieves information about a specific blobber given its id.

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

### <span id="get-blobber-allocations"></span> Get blobber allocations. (*GetBlobberAllocations*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/blobber-allocations
```

Gets a list of allocation information for allocations hosted on a specific blobber. Supports pagination.

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
| [200](#get-blobber-allocations-200) | OK | StorageAllocation |  | [schema](#get-blobber-allocations-200-schema) |
| [400](#get-blobber-allocations-400) | Bad Request |  |  | [schema](#get-blobber-allocations-400-schema) |
| [500](#get-blobber-allocations-500) | Internal Server Error |  |  | [schema](#get-blobber-allocations-500-schema) |

#### Responses


##### <span id="get-blobber-allocations-200"></span> 200 - StorageAllocation
Status: OK

###### <span id="get-blobber-allocations-200-schema"></span> Schema
   
  

[][StorageAllocation](#storage-allocation)

##### <span id="get-blobber-allocations-400"></span> 400
Status: Bad Request

###### <span id="get-blobber-allocations-400-schema"></span> Schema

##### <span id="get-blobber-allocations-500"></span> 500
Status: Internal Server Error

###### <span id="get-blobber-allocations-500-schema"></span> Schema

### <span id="get-blobber-challenges"></span> Get blobber challenges. (*GetBlobberChallenges*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/blobber-challenges
```

Gets list of challenges for a blobber in a specific time interval, given the blobber id.

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| from | `query` | string | `string` |  | ✓ |  | start time of the interval for which to get challenges (epoch timestamp in seconds) |
| id | `query` | string | `string` |  | ✓ |  | id of blobber for which to get challenges |
| to | `query` | string | `string` |  | ✓ |  | end time of interval for which to get challenges (epoch timestamp in seconds) |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#get-blobber-challenges-200) | OK | Challenges |  | [schema](#get-blobber-challenges-200-schema) |
| [400](#get-blobber-challenges-400) | Bad Request |  |  | [schema](#get-blobber-challenges-400-schema) |
| [404](#get-blobber-challenges-404) | Not Found |  |  | [schema](#get-blobber-challenges-404-schema) |
| [500](#get-blobber-challenges-500) | Internal Server Error |  |  | [schema](#get-blobber-challenges-500-schema) |

#### Responses


##### <span id="get-blobber-challenges-200"></span> 200 - Challenges
Status: OK

###### <span id="get-blobber-challenges-200-schema"></span> Schema
   
  


 [Challenges](#challenges)

##### <span id="get-blobber-challenges-400"></span> 400
Status: Bad Request

###### <span id="get-blobber-challenges-400-schema"></span> Schema

##### <span id="get-blobber-challenges-404"></span> 404
Status: Not Found

###### <span id="get-blobber-challenges-404-schema"></span> Schema

##### <span id="get-blobber-challenges-500"></span> 500
Status: Internal Server Error

###### <span id="get-blobber-challenges-500-schema"></span> Schema

### <span id="get-blobber-ids"></span> Get blobber ids by blobber urls. (*GetBlobberIds*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/blobber_ids
```

Returns list of blobber ids given their urls. Supports pagination.

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
| [200](#get-blobber-ids-200) | OK | stringArray |  | [schema](#get-blobber-ids-200-schema) |
| [400](#get-blobber-ids-400) | Bad Request |  |  | [schema](#get-blobber-ids-400-schema) |

#### Responses


##### <span id="get-blobber-ids-200"></span> 200 - stringArray
Status: OK

###### <span id="get-blobber-ids-200-schema"></span> Schema
   
  


 [StringArray](#string-array)

##### <span id="get-blobber-ids-400"></span> 400
Status: Bad Request

###### <span id="get-blobber-ids-400-schema"></span> Schema

### <span id="get-blobbers"></span> Get active blobbers ids. (*GetBlobbers*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/getblobbers
```

Retrieve active blobbers' ids. Retrieved  blobbers should be alive (e.g. excluding blobbers with zero capacity).

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| limit | `query` | string | `string` |  |  |  | limit |
| offset | `query` | string | `string` |  |  |  | offset |
| sort | `query` | string | `string` |  |  |  | desc or asc |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#get-blobbers-200) | OK | storageNodesResponse |  | [schema](#get-blobbers-200-schema) |
| [500](#get-blobbers-500) | Internal Server Error |  |  | [schema](#get-blobbers-500-schema) |

#### Responses


##### <span id="get-blobbers-200"></span> 200 - storageNodesResponse
Status: OK

###### <span id="get-blobbers-200-schema"></span> Schema
   
  

[StorageNodesResponse](#storage-nodes-response)

##### <span id="get-blobbers-500"></span> 500
Status: Internal Server Error

###### <span id="get-blobbers-500-schema"></span> Schema

### <span id="get-block"></span> Gets block information (*GetBlock*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/block
```

Returns block information for a given block hash or block round.

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| block_hash | `query` | string | `string` |  |  |  | Hash (or identifier) of the block |
| date | `query` | string | `string` |  |  |  | block created closest to the date (epoch timestamp in seconds) |
| round | `query` | string | `string` |  |  |  | block round |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#get-block-200) | OK | Block |  | [schema](#get-block-200-schema) |
| [400](#get-block-400) | Bad Request |  |  | [schema](#get-block-400-schema) |
| [500](#get-block-500) | Internal Server Error |  |  | [schema](#get-block-500-schema) |

#### Responses


##### <span id="get-block-200"></span> 200 - Block
Status: OK

###### <span id="get-block-200-schema"></span> Schema
   
  

[Block](#block)

##### <span id="get-block-400"></span> 400
Status: Bad Request

###### <span id="get-block-400-schema"></span> Schema

##### <span id="get-block-500"></span> 500
Status: Internal Server Error

###### <span id="get-block-500-schema"></span> Schema

### <span id="get-blocks"></span> Get blocks for round range. (*GetBlocks*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/get_blocks
```

Gets block information for a list of blocks given a range of block numbers. Supports pagination.

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| end | `query` | string | `string` |  | ✓ |  | last round to get blocks for. |
| limit | `query` | string | `string` |  |  |  | limit |
| offset | `query` | string | `string` |  |  |  | offset |
| sort | `query` | string | `string` |  |  |  | desc or asc |
| start | `query` | string | `string` |  | ✓ |  | first round to get blocks for. |

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

### <span id="get-challenge"></span> Get challenge information. (*GetChallenge*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/getchallenge
```

Returns challenge information given its id.

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

### <span id="get-challenge-pool-stat"></span> Get challenge pool statistics. (*GetChallengePoolStat*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/getChallengePoolStat
```

Retrieve statistic for all locked tokens of a challenge pool.

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

### <span id="get-collected-reward"></span> Get collected reward. (*GetCollectedReward*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/collected_reward
```

Returns collected reward for a client_id.

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
| [200](#get-collected-reward-200) | OK | challengePoolStat |  | [schema](#get-collected-reward-200-schema) |
| [400](#get-collected-reward-400) | Bad Request |  |  | [schema](#get-collected-reward-400-schema) |

#### Responses


##### <span id="get-collected-reward-200"></span> 200 - challengePoolStat
Status: OK

###### <span id="get-collected-reward-200-schema"></span> Schema
   
  

[ChallengePoolStat](#challenge-pool-stat)

##### <span id="get-collected-reward-400"></span> 400
Status: Bad Request

###### <span id="get-collected-reward-400-schema"></span> Schema

### <span id="get-expired-allocations"></span> Get expired allocations. (*GetExpiredAllocations*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/getExpiredAllocations
```

Retrieves a list of expired allocations associated with a specified blobber.

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

### <span id="get-free-alloc-blobbers"></span> Get free allocation blobbers. (*GetFreeAllocBlobbers*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/free_alloc_blobbers
```

Returns a list of all active blobbers that match the free allocation request.

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
| [200](#get-free-alloc-blobbers-200) | OK | stringArray |  | [schema](#get-free-alloc-blobbers-200-schema) |
| [400](#get-free-alloc-blobbers-400) | Bad Request |  |  | [schema](#get-free-alloc-blobbers-400-schema) |

#### Responses


##### <span id="get-free-alloc-blobbers-200"></span> 200 - stringArray
Status: OK

###### <span id="get-free-alloc-blobbers-200-schema"></span> Schema
   
  


 [StringArray](#string-array)

##### <span id="get-free-alloc-blobbers-400"></span> 400
Status: Bad Request

###### <span id="get-free-alloc-blobbers-400-schema"></span> Schema

### <span id="get-latest-readmarker"></span> Get latest read marker. (*GetLatestReadmarker*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/latestreadmarker
```

Retrievs latest read marker for a client and a blobber.

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| allocation | `query` | string | `string` |  |  |  | Allocation ID associated with the read marker. |
| blobber | `query` | string | `string` |  | ✓ |  | blobber ID associated with the read marker. |
| client | `query` | string | `string` |  | ✓ |  | ID of the client for which to get the latest read marker. |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#get-latest-readmarker-200) | OK | ReadMarker |  | [schema](#get-latest-readmarker-200-schema) |
| [500](#get-latest-readmarker-500) | Internal Server Error |  |  | [schema](#get-latest-readmarker-500-schema) |

#### Responses


##### <span id="get-latest-readmarker-200"></span> 200 - ReadMarker
Status: OK

###### <span id="get-latest-readmarker-200-schema"></span> Schema
   
  

[ReadMarker](#read-marker)

##### <span id="get-latest-readmarker-500"></span> 500
Status: Internal Server Error

###### <span id="get-latest-readmarker-500-schema"></span> Schema

### <span id="get-open-challenges"></span> Get blobber open challenges. (*GetOpenChallenges*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/openchallenges
```

Retrieves open challenges for a blobber given its id.

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
| [200](#get-open-challenges-200) | OK | ChallengesResponse |  | [schema](#get-open-challenges-200-schema) |
| [400](#get-open-challenges-400) | Bad Request |  |  | [schema](#get-open-challenges-400-schema) |
| [404](#get-open-challenges-404) | Not Found |  |  | [schema](#get-open-challenges-404-schema) |
| [500](#get-open-challenges-500) | Internal Server Error |  |  | [schema](#get-open-challenges-500-schema) |

#### Responses


##### <span id="get-open-challenges-200"></span> 200 - ChallengesResponse
Status: OK

###### <span id="get-open-challenges-200-schema"></span> Schema
   
  

[ChallengesResponse](#challenges-response)

##### <span id="get-open-challenges-400"></span> 400
Status: Bad Request

###### <span id="get-open-challenges-400-schema"></span> Schema

##### <span id="get-open-challenges-404"></span> 404
Status: Not Found

###### <span id="get-open-challenges-404-schema"></span> Schema

##### <span id="get-open-challenges-500"></span> 500
Status: Internal Server Error

###### <span id="get-open-challenges-500-schema"></span> Schema

### <span id="get-read-markers"></span> Get read markers. (*GetReadMarkers*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/readmarkers
```

Retrieves read markers given an allocation id or an auth ticket. Supports pagination.

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| allocation_id | `query` | string | `string` |  |  |  | filter in only read markers by this allocation. Either this or auth_ticket must be provided. |
| auth_ticket | `query` | string | `string` |  |  |  | filter in only read markers using this auth ticket. Either this or allocation_id must be provided. |
| limit | `query` | string | `string` |  |  |  | limit |
| offset | `query` | string | `string` |  |  |  | offset |
| sort | `query` | string | `string` |  |  |  | desc or asc |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#get-read-markers-200) | OK | ReadMarker |  | [schema](#get-read-markers-200-schema) |
| [500](#get-read-markers-500) | Internal Server Error |  |  | [schema](#get-read-markers-500-schema) |

#### Responses


##### <span id="get-read-markers-200"></span> 200 - ReadMarker
Status: OK

###### <span id="get-read-markers-200-schema"></span> Schema
   
  

[][ReadMarker](#read-marker)

##### <span id="get-read-markers-500"></span> 500
Status: Internal Server Error

###### <span id="get-read-markers-500-schema"></span> Schema

### <span id="get-read-markers-count"></span> Gets read markers count. (*GetReadMarkersCount*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/count_readmarkers
```

Returns the count of read markers for a given allocation.

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| allocation_id | `query` | string | `string` |  | ✓ |  | count read markers for this allocation |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#get-read-markers-count-200) | OK | readMarkersCount |  | [schema](#get-read-markers-count-200-schema) |
| [500](#get-read-markers-count-500) | Internal Server Error |  |  | [schema](#get-read-markers-count-500-schema) |

#### Responses


##### <span id="get-read-markers-count-200"></span> 200 - readMarkersCount
Status: OK

###### <span id="get-read-markers-count-200-schema"></span> Schema
   
  

[ReadMarkersCount](#read-markers-count)

##### <span id="get-read-markers-count-500"></span> 500
Status: Internal Server Error

###### <span id="get-read-markers-count-500-schema"></span> Schema

### <span id="get-read-pool-stat"></span> Get read pool statistics. (*GetReadPoolStat*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/getReadPoolStat
```

Retrieve statistic for all locked tokens of the read pool of a client given their id.

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

### <span id="get-stake-pool-stat"></span> Get stake pool statistics. (*GetStakePoolStat*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/getStakePoolStat
```

Retrieve statistic for all locked tokens of a stake pool associated with a specific client and provider. Provider can be a blobber, validator, or authorizer.

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

### <span id="get-storage-config"></span> Get storage smart contract settings. (*GetStorageConfig*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/storage-config
```

Retrieve the current storage smart contract settings.

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#get-storage-config-200) | OK | StringMap |  | [schema](#get-storage-config-200-schema) |
| [400](#get-storage-config-400) | Bad Request |  |  | [schema](#get-storage-config-400-schema) |

#### Responses


##### <span id="get-storage-config-200"></span> 200 - StringMap
Status: OK

###### <span id="get-storage-config-200-schema"></span> Schema
   
  

[StringMap](#string-map)

##### <span id="get-storage-config-400"></span> 400
Status: Bad Request

###### <span id="get-storage-config-400-schema"></span> Schema

### <span id="get-transaction"></span> Get transaction information (*GetTransaction*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/transaction
```

Gets transaction information given transaction hash.

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| transaction_hash | `query` | string | `string` |  | ✓ |  | The hash of the transaction to retrieve. |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#get-transaction-200) | OK | Transaction |  | [schema](#get-transaction-200-schema) |
| [500](#get-transaction-500) | Internal Server Error |  |  | [schema](#get-transaction-500-schema) |

#### Responses


##### <span id="get-transaction-200"></span> 200 - Transaction
Status: OK

###### <span id="get-transaction-200-schema"></span> Schema
   
  

[Transaction](#transaction)

##### <span id="get-transaction-500"></span> 500
Status: Internal Server Error

###### <span id="get-transaction-500-schema"></span> Schema

### <span id="get-transaction-errors"></span> Get transaction errors. (*GetTransactionErrors*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/errors
```

Retrieves a list of errors associated with a specific transaction. Supports pagination.

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
| [200](#get-transaction-errors-200) | OK | Error |  | [schema](#get-transaction-errors-200-schema) |
| [400](#get-transaction-errors-400) | Bad Request |  |  | [schema](#get-transaction-errors-400-schema) |
| [500](#get-transaction-errors-500) | Internal Server Error |  |  | [schema](#get-transaction-errors-500-schema) |

#### Responses


##### <span id="get-transaction-errors-200"></span> 200 - Error
Status: OK

###### <span id="get-transaction-errors-200-schema"></span> Schema
   
  

[][Error](#error)

##### <span id="get-transaction-errors-400"></span> 400
Status: Bad Request

###### <span id="get-transaction-errors-400-schema"></span> Schema

##### <span id="get-transaction-errors-500"></span> 500
Status: Internal Server Error

###### <span id="get-transaction-errors-500-schema"></span> Schema

### <span id="get-transactions"></span> Get Transactions	list. (*GetTransactions*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/transactions
```

Gets filtered list of transaction information. The list is filtered on the first valid input, or otherwise all the endpoint returns all translations.

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
| [200](#get-transactions-200) | OK | Transaction |  | [schema](#get-transactions-200-schema) |
| [400](#get-transactions-400) | Bad Request |  |  | [schema](#get-transactions-400-schema) |
| [500](#get-transactions-500) | Internal Server Error |  |  | [schema](#get-transactions-500-schema) |

#### Responses


##### <span id="get-transactions-200"></span> 200 - Transaction
Status: OK

###### <span id="get-transactions-200-schema"></span> Schema
   
  

[][Transaction](#transaction)

##### <span id="get-transactions-400"></span> 400
Status: Bad Request

###### <span id="get-transactions-400-schema"></span> Schema

##### <span id="get-transactions-500"></span> 500
Status: Internal Server Error

###### <span id="get-transactions-500-schema"></span> Schema

### <span id="get-user-stake-pool-stat"></span> Get user stake pool statistics. (*GetUserStakePoolStat*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/getUserStakePoolStat
```

Retrieve statistic for a user's stake pools given the user's id.

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

### <span id="get-validator"></span> Get validator information. (*GetValidator*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/get_validator
```

Retrieve information for a validator given its id.

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

### <span id="get-validators"></span> Get validators. (*GetValidators*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/validators
```

Retrieves a list of validators, optionally filtered by whether they are active and/or stakable.

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
| [200](#get-validators-200) | OK | validatorNodeResponse |  | [schema](#get-validators-200-schema) |
| [400](#get-validators-400) | Bad Request |  |  | [schema](#get-validators-400-schema) |

#### Responses


##### <span id="get-validators-200"></span> 200 - validatorNodeResponse
Status: OK

###### <span id="get-validators-200-schema"></span> Schema
   
  

[][ValidatorNodeResponse](#validator-node-response)

##### <span id="get-validators-400"></span> 400
Status: Bad Request

###### <span id="get-validators-400-schema"></span> Schema

### <span id="get-write-markers"></span> Get write markers. (*GetWriteMarkers*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/writemarkers
```

Retrieves a list of write markers satisfying filter. Supports pagination.

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| limit | `query` | string | `string` |  |  |  | limit |
| offset | `query` | string | `string` |  |  |  | offset |
| sort | `query` | string | `string` |  |  |  | asc or desc |

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

Generic search endpoint that can be used to search for blocks, transactions, users, etc.

If the input can be converted to an integer, it is interpreted as a round number and information for the matching block is returned.

Otherwise, the input is treated as string and matched against block hash, transaction hash, user id. If a match is found the matching object is returned.

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

### <span id="challenge-pool-stat"></span> challengePoolStat


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| Finalized | boolean| `bool` |  | |  |  |
| ID | string| `string` |  | |  |  |
| balance | [Coin](#coin)| `Coin` |  | |  |  |
| expiration | [Timestamp](#timestamp)| `Timestamp` |  | |  |  |
| start_time | [Timestamp](#timestamp)| `Timestamp` |  | |  |  |



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


