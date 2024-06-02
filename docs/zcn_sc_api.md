


# ZCN Smart Contract Public API:
  

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
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712e0/getAuthorizer | [get authorizer](#get-authorizer) | Get authorizer. |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712e0/getAuthorizerNodes | [get authorizer nodes](#get-authorizer-nodes) | Get authorizer nodes. |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712e0/getGlobalConfig | [get global config](#get-global-config) | Get smart contract configuration. |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712e0/v1/mint_nonce | [get mint nonce](#get-mint-nonce) | Get mint nonce. |
| GET | /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712e0/v1/not_processed_burn_tickets | [get not processed burn tickets](#get-not-processed-burn-tickets) | Get not processed burn tickets. |
  


## Paths

### <span id="get-authorizer"></span> Get authorizer. (*GetAuthorizer*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712e0/getAuthorizer
```

Retrieve details of an authorizer given its ID.

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| id | `query` | string | `string` |  | ✓ |  | "Authorizer ID" |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#get-authorizer-200) | OK | authorizerResponse |  | [schema](#get-authorizer-200-schema) |
| [400](#get-authorizer-400) | Bad Request |  |  | [schema](#get-authorizer-400-schema) |
| [404](#get-authorizer-404) | Not Found |  |  | [schema](#get-authorizer-404-schema) |

#### Responses


##### <span id="get-authorizer-200"></span> 200 - authorizerResponse
Status: OK

###### <span id="get-authorizer-200-schema"></span> Schema
   
  

[AuthorizerResponse](#authorizer-response)

##### <span id="get-authorizer-400"></span> 400
Status: Bad Request

###### <span id="get-authorizer-400-schema"></span> Schema

##### <span id="get-authorizer-404"></span> 404
Status: Not Found

###### <span id="get-authorizer-404-schema"></span> Schema

### <span id="get-authorizer-nodes"></span> Get authorizer nodes. (*GetAuthorizerNodes*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712e0/getAuthorizerNodes
```

Retrieve the list of authorizer nodes.

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| active | `query` | boolean | `bool` |  |  |  | "If true, returns only active authorizers" |

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

### <span id="get-global-config"></span> Get smart contract configuration. (*GetGlobalConfig*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712e0/getGlobalConfig
```

Retrieve the smart contract configuration in JSON format.

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

### <span id="get-mint-nonce"></span> Get mint nonce. (*GetMintNonce*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712e0/v1/mint_nonce
```

Retrieve the latest mint nonce for the client with the given client ID.

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| client_id | `query` | string | `string` |  | ✓ |  | "Client ID" |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#get-mint-nonce-200) | OK | Int64Map |  | [schema](#get-mint-nonce-200-schema) |
| [400](#get-mint-nonce-400) | Bad Request |  |  | [schema](#get-mint-nonce-400-schema) |
| [404](#get-mint-nonce-404) | Not Found |  |  | [schema](#get-mint-nonce-404-schema) |

#### Responses


##### <span id="get-mint-nonce-200"></span> 200 - Int64Map
Status: OK

###### <span id="get-mint-nonce-200-schema"></span> Schema
   
  

[Int64Map](#int64-map)

##### <span id="get-mint-nonce-400"></span> 400
Status: Bad Request

###### <span id="get-mint-nonce-400-schema"></span> Schema

##### <span id="get-mint-nonce-404"></span> 404
Status: Not Found

###### <span id="get-mint-nonce-404-schema"></span> Schema

### <span id="get-not-processed-burn-tickets"></span> Get not processed burn tickets. (*GetNotProcessedBurnTickets*)

```
GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712e0/v1/not_processed_burn_tickets
```

Retrieve the not processed ZCN burn tickets for the given ethereum address and client id with a help of offset nonce.
The burn tickets are returned in ascending order of nonce. Only burn tickets with nonce greater than the given nonce are returned.
This is an indicator of the burn tickets that are not processed yet after the given nonce. If nonce is not provided, all un-processed burn tickets are returned.

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| ethereum_address | `query` | string | `string` |  | ✓ |  | "Ethereum address" |
| nonce | `query` | string | `string` |  |  |  | "Offset nonce" |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#get-not-processed-burn-tickets-200) | OK | BurnTicket |  | [schema](#get-not-processed-burn-tickets-200-schema) |
| [400](#get-not-processed-burn-tickets-400) | Bad Request |  |  | [schema](#get-not-processed-burn-tickets-400-schema) |

#### Responses


##### <span id="get-not-processed-burn-tickets-200"></span> 200 - BurnTicket
Status: OK

###### <span id="get-not-processed-burn-tickets-200-schema"></span> Schema
   
  

[][BurnTicket](#burn-ticket)

##### <span id="get-not-processed-burn-tickets-400"></span> 400
Status: Bad Request

###### <span id="get-not-processed-burn-tickets-400-schema"></span> Schema

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



### <span id="burn-ticket"></span> BurnTicket


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| EthereumAddress | string| `string` |  | |  |  |
| Hash | string| `string` |  | |  |  |
| Nonce | int64 (formatted integer)| `int64` |  | |  |  |
| amount | [Coin](#coin)| `Coin` |  | |  |  |



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



### <span id="provider"></span> Provider


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| CreatedAt | date-time (formatted string)| `strfmt.DateTime` |  | |  |  |
| DelegateWallet | string| `string` |  | |  |  |
| Downtime | uint64 (formatted integer)| `uint64` |  | |  |  |
| ID | string| `string` |  | |  |  |
| IsKilled | boolean| `bool` |  | |  |  |
| IsShutdown | boolean| `bool` |  | |  |  |
| NumDelegates | int64 (formatted integer)| `int64` |  | |  |  |
| ServiceCharge | double (formatted number)| `float64` |  | |  |  |
| UpdatedAt | date-time (formatted string)| `strfmt.DateTime` |  | |  |  |
| last_health_check | [Timestamp](#timestamp)| `Timestamp` |  | |  |  |
| rewards | [ProviderRewards](#provider-rewards)| `ProviderRewards` |  | |  |  |
| total_stake | [Coin](#coin)| `Coin` |  | |  |  |



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


