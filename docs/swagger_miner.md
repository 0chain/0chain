


# Miner API:
  

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
| POST | /v1/transaction/put | [put transaction](#put-transaction) | PutTransaction - Put a transaction to the transaction pool. Transaction size cannot exceed the max payload size which is a global configuration of the chain. |
| GET | /v1/chain/get/stats | [chainstatus](#chainstatus) |  |
  


## Paths

### <span id="put-transaction"></span> PutTransaction - Put a transaction to the transaction pool. Transaction size cannot exceed the max payload size which is a global configuration of the chain. (*PutTransaction*)

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

### <span id="chainstatus"></span> chainstatus (*chainstatus*)

```
GET /v1/chain/get/stats
```

a handler to provide block statistics

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#chainstatus-200) | OK | ChainStats |  | [schema](#chainstatus-200-schema) |
| [500](#chainstatus-500) | Internal Server Error |  |  | [schema](#chainstatus-500-schema) |

#### Responses


##### <span id="chainstatus-200"></span> 200 - ChainStats
Status: OK

###### <span id="chainstatus-200-schema"></span> Schema
   
  

[ChainStats](#chain-stats)

##### <span id="chainstatus-500"></span> 500
Status: Internal Server Error

###### <span id="chainstatus-500-schema"></span> Schema

## Models

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



### <span id="client"></span> Client


> go:generate msgp -io=false -tests=false -v
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| ID | string| `string` |  | |  |  |
| PublicKey | string| `string` |  | |  |  |
| Version | string| `string` |  | |  |  |
| creation_date | [Timestamp](#timestamp)| `Timestamp` |  | |  |  |



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



### <span id="duration"></span> Duration


> A Duration represents the elapsed time between two instants
as an int64 nanosecond count. The representation limits the
largest representable duration to approximately 290 years.
  



| Name | Type | Go type | Default | Description | Example |
|------|------|---------| ------- |-------------|---------|
| Duration | int64 (formatted integer)| int64 | | A Duration represents the elapsed time between two instants</br>as an int64 nanosecond count. The representation limits the</br>largest representable duration to approximately 290 years. |  |



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



### <span id="share-or-signs"></span> ShareOrSigns


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| ID | string| `string` |  | |  |  |
| ShareOrSigns | map of [DKGKeyShare](#d-k-g-key-share)| `map[string]DKGKeyShare` |  | |  |  |



### <span id="timestamp"></span> Timestamp


> go:generate msgp -io=false -tests=false -v
Timestamp - just a wrapper to control the json encoding */
  



| Name | Type | Go type | Default | Description | Example |
|------|------|---------| ------- |-------------|---------|
| Timestamp | int64 (formatted integer)| int64 | | go:generate msgp -io=false -tests=false -v</br>Timestamp - just a wrapper to control the json encoding */ |  |



### <span id="version-field"></span> VersionField


> go:generate msgp -io=false -tests=false -v
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| Version | string| `string` |  | |  |  |


