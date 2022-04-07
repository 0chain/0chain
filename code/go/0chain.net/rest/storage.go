package rest

import (
	"net/http"
	"strconv"

	"0chain.net/chaincore/state"

	"0chain.net/smartcontract/stakepool/spenum"

	"0chain.net/smartcontract/dbs/event"

	"0chain.net/core/datastore"
	"0chain.net/core/util"

	"0chain.net/smartcontract"

	"0chain.net/smartcontract/storagesc"

	"0chain.net/core/common"
)

type StorageRestHandler struct {
	*RestHandler
}

func NewStorageRestHandler(rh *RestHandler) *StorageRestHandler {
	return &StorageRestHandler{rh}
}

func SetupStorageRestHandler(rh *RestHandler) {
	srh := NewStorageRestHandler(rh)
	storage := "/v1/screst/" + storagesc.ADDRESS
	http.HandleFunc(storage+"/get_blobber_count", srh.getBlobberCount)
	http.HandleFunc(storage+"/getBlobber", srh.getBlobber)
	http.HandleFunc(storage+"/getblobbers", srh.getBlobbers)
	http.HandleFunc(storage+"/get_blobber_total_stakes", srh.getBlobberTotalStakes)
	http.HandleFunc(storage+"/get_blobber_lat_long", srh.getBlobberGeoLocation)
	http.HandleFunc(storage+"/transaction", srh.getTransactionByHash)
	http.HandleFunc(storage+"/transactions", srh.getTransactionByFilter)
	http.HandleFunc(storage+"/writemarkers", srh.getWriteMarker)
	http.HandleFunc(storage+"/errors", srh.getErrors)
	//http.HandleFunc(storage+"/allocations", srh.getAllocations)
	//http.HandleFunc(storage+"/allocation_min_lock", srh.getAllocationMinLock)
	http.HandleFunc(storage+"/allocation", srh.getAllocationStats)
	http.HandleFunc(storage+"/latestreadmarker", srh.getLatestReadMarker)
	http.HandleFunc(storage+"/readmarkers", srh.getReadMarkers)
	http.HandleFunc(storage+"/count_readmarkers", srh.getReadMarkersCount)
	http.HandleFunc(storage+"/getWriteMarkers", srh.getWriteMarkers)
	http.HandleFunc(storage+"/get_validator", srh.getValidator)
	http.HandleFunc(storage+"/openchallenges", srh.getOpenChallenges)
	http.HandleFunc(storage+"/getchallenge", srh.getChallenge)
	http.HandleFunc(storage+"/getStakePoolStat", srh.getStakePoolStat)
	http.HandleFunc(storage+"/getUserStakePoolStat", srh.getUserStakePoolStat)
	http.HandleFunc(storage+"/get_block_by_hash", srh.getBlockByHash)
	http.HandleFunc(storage+"/get_blocks", srh.getBlocks)
	http.HandleFunc(storage+"/total_saved_data", srh.getTotalData)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/get_blocks get_blocks
// Gets the total data stored across all blobbers. Todo: We need to rewrite this to use event database not MPT
//
// responses:
//  200: int64Map
//  400:
func (_ *StorageRestHandler) getTotalData(w http.ResponseWriter, r *http.Request) {
	common.Respond(w, r, nil, common.NewErrInternal("not implemented yet"))
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/get_blocks get_blocks
// Gets block information for all blocks. Todo: We need to add a filter to this.
//
// parameters:
//    + name: block_hash
//      description: block hash
//      required: true
//      in: query
//      type: string
//
// responses:
//  200: userPoolStat
//  400:
//  500:
func (srh *StorageRestHandler) getBlocks(w http.ResponseWriter, r *http.Request) {
	block, err := srh.GetEventDB().GetBlocks()
	if err != nil {
		common.Respond(w, r, nil, common.NewErrInternal("getting block "+err.Error()))
		return
	}
	common.Respond(w, r, &block, nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/getUserStakePoolStat getUserStakePoolStat
// Gets block information from block hash
//
// parameters:
//    + name: block_hash
//      description: block hash
//      required: true
//      in: query
//      type: string
//
// responses:
//  200: userPoolStat
//  400:
//  500:
func (srh *StorageRestHandler) getBlockByHash(w http.ResponseWriter, r *http.Request) {
	hash := r.URL.Query().Get("block_hash")
	if len(hash) == 0 {
		common.Respond(w, r, nil, common.NewErrBadRequest("annot find valid block hash: "+hash))
		return
	}

	block, err := srh.GetEventDB().GetBlocksByHash(hash)
	if err != nil {
		common.Respond(w, r, nil, common.NewErrInternal("getting block "+err.Error()))
		return
	}

	common.Respond(w, r, &block, nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/getUserStakePoolStat getUserStakePoolStat
// Gets statistic for a user's stake pools
//
// parameters:
//    + name: client_id
//      description: client for which to get stake pool information
//      required: true
//      in: query
//      type: string
//
// responses:
//  200: userPoolStat
//  400:
func (srh *StorageRestHandler) getUserStakePoolStat(w http.ResponseWriter, r *http.Request) {
	clientID := r.URL.Query().Get("client_id")

	pools, err := srh.GetEventDB().GetUserDelegatePools(clientID, int(spenum.Blobber))
	if err != nil {
		common.Respond(w, r, nil, common.NewErrBadRequest("blobber not found in event database: "+err.Error()))
		return
	}

	var ups = new(userPoolStat)
	ups.Pools = make(map[datastore.Key][]*storagesc.DelegatePoolStat)
	for _, pool := range pools {
		var dps = storagesc.DelegatePoolStat{
			ID:           pool.PoolID,
			Balance:      state.Balance(pool.Balance),
			DelegateID:   pool.DelegateID,
			Rewards:      state.Balance(pool.Reward),
			TotalPenalty: state.Balance(pool.TotalPenalty),
			TotalReward:  state.Balance(pool.TotalReward),
			Status:       spenum.PoolStatus(pool.Status).String(),
			RoundCreated: pool.RoundCreated,
		}
		ups.Pools[pool.ProviderID] = append(ups.Pools[pool.ProviderID], &dps)
	}

	common.Respond(w, r, ups, nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/getStakePoolStat getStakePoolStat
// Gets statistic for all locked tokens of a stake pool
//
// parameters:
//    + name: blobber_id
//      description: id of blobber
//      required: true
//      in: query
//      type: string
//
// responses:
//  200: stakePoolStat
//  400:
//  500:
func (srh *StorageRestHandler) getStakePoolStat(w http.ResponseWriter, r *http.Request) {
	blobberID := r.URL.Query().Get("blobber_id")

	blobber, err := srh.GetEventDB().GetBlobber(blobberID)
	if err != nil {
		common.Respond(w, r, nil, common.NewErrBadRequest("cannot find blobber: "+err.Error()))
		return
	}

	delegatePools, err := srh.GetEventDB().GetDelegatePools(blobberID, int(spenum.Blobber))
	if err != nil {
		common.Respond(w, r, nil, common.NewErrInternal("cannot find user stake pool: "+err.Error()))
		return
	}
	common.Respond(w, r, spStats(*blobber, delegatePools), nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/getchallenge getchallenge
// Gets challenges for a blobber by challenge id
//
// parameters:
//    + name: blobber
//      description: id of blobber
//      required: true
//      in: query
//      type: string
//    + name: challenge
//      description: id of challenge
//      required: true
//      in: query
//      type: string
//
// responses:
//  200: StorageChallenge
//  400:
//  404:
//  500:
func (srh *StorageRestHandler) getChallenge(w http.ResponseWriter, r *http.Request) {
	blobberID := r.URL.Query().Get("blobber")
	blobberChallengeObj := &storagesc.BlobberChallenge{}
	blobberChallengeObj.BlobberID = blobberID
	blobberChallengeObj.ChallengeIDs = make([]string, 0)

	err := srh.GetTrieNode(blobberChallengeObj.GetKey(storagesc.ADDRESS), blobberChallengeObj)
	if err != nil {
		common.Respond(w, r, nil, smartcontract.NewErrNoResourceOrErrInternal(err, true, "can't get blobber challenge"))
		return
	}

	challengeID := r.URL.Query().Get("challenge")
	if _, ok := blobberChallengeObj.ChallengeIDMap[challengeID]; !ok {
		common.Respond(w, r, nil, common.NewErrBadRequest("can't find challenge with provided 'challenge' param"))
		return
	}

	challenge := new(storagesc.StorageChallenge)
	challenge.ID = challengeID
	err = srh.GetTrieNode(challenge.GetKey(storagesc.ADDRESS), challenge)
	if err != nil {
		common.Respond(w, r, nil, smartcontract.NewErrNoResourceOrErrInternal(err, true, "can't get storage challenge"))
		return
	}

	common.Respond(w, r, challenge, nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/openchallenges openchallenges
// Gets open challenges for a blobber
//
// parameters:
//    + name: blobber
//      description: id of blobber for which to get open challenges
//      required: true
//      in: query
//      type: string
//
// responses:
//  200: BlobberChallenge
//  400:
//  404:
//  500:
func (srh *StorageRestHandler) getOpenChallenges(w http.ResponseWriter, r *http.Request) {
	blobberID := r.URL.Query().Get("blobber")
	if blobberID == "" {
		common.Respond(w, r, nil, common.NewErrBadRequest("no blobber id"))
		return
	}

	// return "404", if blobber not registered
	blobber := storagesc.StorageNode{ID: blobberID}
	if err := srh.GetTrieNode(blobber.GetKey(storagesc.ADDRESS), &blobber); err != nil {
		common.Respond(w, r, nil, smartcontract.NewErrNoResourceOrErrInternal(err, true, "can't find blobber"))
		return
	}

	// return "200" with empty list, if no challenges are found
	blobberChallengeObj := &storagesc.BlobberChallenge{BlobberID: blobberID}
	blobberChallengeObj.ChallengeIDs = make([]string, 0)
	err := srh.GetTrieNode(blobberChallengeObj.GetKey(storagesc.ADDRESS), blobberChallengeObj)
	switch err {
	case nil, util.ErrValueNotPresent:
		common.Respond(w, r, blobberChallengeObj, nil)
	default:
		common.Respond(w, r, nil, common.NewErrInternal("fail to get blobber challenge", err.Error()))
	}
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/get_validator get_validator
// Gets validator information
//
// parameters:
//    + name: validator_id
//      description: validator on which to get information
//      required: true
//      in: query
//      type: string
//
// responses:
//  200: []Validator
//  400:
//  500:
func (srh *StorageRestHandler) getValidator(w http.ResponseWriter, r *http.Request) {

	var (
		validatorID = r.URL.Query().Get("validator_id")
	)

	if validatorID == "" {
		common.Respond(w, r, nil, common.NewErrBadRequest("no validator id"))
		return
	}

	validator, err := srh.GetEventDB().GetValidatorByValidatorID(validatorID)
	if err != nil {
		common.Respond(w, r, nil, common.NewErrInternal("can't find validator", err.Error()))
		return
	}

	common.Respond(w, r, validator, nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/getWriteMarkers getWriteMarkers
// Gets read markers according to a filter
//
// parameters:
//    + name: allocation_id
//      description: count write markers for this allocation
//      required: true
//      in: query
//      type: string
//
// responses:
//  200: []WriteMarker
//  400:
//  500:
func (srh *StorageRestHandler) getWriteMarkers(w http.ResponseWriter, r *http.Request) {
	var (
		allocationID = r.URL.Query().Get("allocation_id")
	)

	if allocationID == "" {
		common.Respond(w, r, nil, common.NewErrBadRequest("no allocation id"))
		return
	}

	writeMarkers, err := srh.GetEventDB().GetWriteMarkersForAllocationID(allocationID)
	if err != nil {
		common.Respond(w, r, nil, common.NewErrInternal("can't count write markers", err.Error()))
		return
	}

	common.Respond(w, r, writeMarkers, nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/count_readmarkers count_readmarkers
// Gets read markers according to a filter
//
// parameters:
//    + name: allocation_id
//      description: count read markers for this allocation
//      required: true
//      in: query
//      type: string
//
// responses:
//  200: readMarkersCount
//  400
//  500:
func (srh *StorageRestHandler) getReadMarkersCount(w http.ResponseWriter, r *http.Request) {
	var (
		allocationID = r.URL.Query().Get("allocation_id")
	)

	if allocationID == "" {
		common.Respond(w, r, nil, common.NewErrBadRequest("no allocation id"))
		return
	}

	query := new(event.ReadMarker)
	if allocationID != "" {
		query.AllocationID = allocationID
	}

	count, err := srh.GetEventDB().CountReadMarkersFromQuery(query)
	if err != nil {
		common.Respond(w, r, nil, common.NewErrInternal("can't count read markers", err.Error()))
		return
	}

	common.Respond(w, r, readMarkersCount{ReadMarkersCount: count}, nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/readmarkers readmarkers
// Gets read markers according to a filter
//
// parameters:
//    + name: allocation_id
//      description: filter read markers by this allocation
//      in: query
//      type: string
//    + name: auth_ticket
//      description: filter in only read markers using auth thicket
//      in: query
//      type: string
//    + name: offset
//      description: offset
//      in: query
//      type: string
//    + name: limit
//      description: limit
//      in: query
//      type: string
//    + name: sort
//      description: desc or asc
//      in: query
//      type: string
//
// responses:
//  200: []ReadMarker
//  500:
func (srh *StorageRestHandler) getReadMarkers(w http.ResponseWriter, r *http.Request) {
	var (
		allocationID = r.URL.Query().Get("allocation_id")
		authTicket   = r.URL.Query().Get("auth_ticket")
		offsetString = r.URL.Query().Get("offset")
		limitString  = r.URL.Query().Get("limit")
		sortString   = r.URL.Query().Get("sort")
		limit        = 0
		offset       = 0
		isDescending = false
	)

	query := event.ReadMarker{}
	if allocationID != "" {
		query.AllocationID = allocationID
	}

	if authTicket != "" {
		query.AuthTicket = authTicket
	}

	if offsetString != "" {
		o, err := strconv.Atoi(offsetString)
		if err != nil {
			common.Respond(w, r, nil, common.NewErrBadRequest("offset is invalid: "+err.Error()))
			return
		}
		offset = o
	}

	if limitString != "" {
		l, err := strconv.Atoi(limitString)
		if err != nil {
			common.Respond(w, r, nil, common.NewErrBadRequest("limit is invalid: "+err.Error()))
			return
		}
		limit = l
	}

	if sortString != "" {
		switch sortString {
		case "desc":
			isDescending = true
		case "asc":
			isDescending = false
		default:
			common.Respond(w, r, nil, common.NewErrBadRequest("sort is invalid: "+sortString))
			return
		}
	}

	readMarkers, err := srh.GetEventDB().GetReadMarkersFromQueryPaginated(query, offset, limit, isDescending)
	if err != nil {
		common.Respond(w, r, nil, common.NewErrInternal("can't get read markers", err.Error()))
		return
	}

	common.Respond(w, r, readMarkers, nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/allocation allocation
// Gets latest read marker for a client and blobber
//
// parameters:
//    + name: client
//      description: client
//      in: query
//      type: string
//    + name: blobber
//      description: blobber
//      in: query
//      type: string
//
// responses:
//  200: []Error
//  500:
func (srh *StorageRestHandler) getLatestReadMarker(w http.ResponseWriter, r *http.Request) {
	var (
		clientID  = r.URL.Query().Get("client")
		blobberID = r.URL.Query().Get("blobber")

		commitRead = &storagesc.ReadConnection{}
	)

	commitRead.ReadMarker = &storagesc.ReadMarker{
		BlobberID: blobberID,
		ClientID:  clientID,
	}

	err := srh.GetTrieNode(commitRead.GetKey(storagesc.ADDRESS), commitRead)
	switch err {
	case nil:
		common.Respond(w, r, commitRead.ReadMarker, nil)
	case util.ErrValueNotPresent:
		common.Respond(w, r, make(map[string]string), nil)
	default:
		common.Respond(w, r, nil, common.NewErrInternal("can't get read marker", err.Error()))
	}
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/allocation_min_lock allocation_min_lock
// Calculates the cost of a new allocation request. Todo redo with changes to new allocation request smart contract
//
// parameters:
//
// responses:
//  200: int64Map
//  400:
//  500:
func (srh *StorageRestHandler) getAllocationMinLock(w http.ResponseWriter, r *http.Request) {
	//var ssc = storagesc.StorageSmartContract{
	//	SmartContract: sci.NewSC(storagesc.ADDRESS),
	//}
	//result, err := ssc.GetAllocationMinLockHandler(r.Context(), r.URL.Query(), )

	common.Respond(w, r, nil, common.NewErrInternal("allocation_min_lock temporary unimplemented"))
}

// getErrors swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/allocation allocation
// Gets allocation object
//
// parameters:
//    + name: transaction_hash
//      description: offset
//      required: true
//      in: query
//      type: string
//
// responses:
//  200: StorageAllocation
//  400:
//  500:
func (srh *StorageRestHandler) getAllocationStats(w http.ResponseWriter, r *http.Request) {
	allocationID := r.URL.Query().Get("allocation")
	allocationObj := &storagesc.StorageAllocation{}
	allocationObj.ID = allocationID

	err := srh.GetTrieNode(allocationObj.GetKey(storagesc.ADDRESS), allocationObj)
	if err != nil {
		common.Respond(w, r, nil, smartcontract.NewErrNoResourceOrErrInternal(err, true, "can't get allocation"))
		return
	}
	common.Respond(w, r, allocationObj, nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/errors errors
// Gets errors returned by indicated transaction
//
// parameters:
//    + name: transaction_hash
//      description: offset
//      required: true
//      in: query
//      type: string
//
// responses:
//  200: []Error
//  400:
//  500:
func (srh *StorageRestHandler) getErrors(w http.ResponseWriter, r *http.Request) {
	transactionHash := r.URL.Query().Get("transaction_hash")
	if len(transactionHash) == 0 {
		common.Respond(w, r, nil, common.NewErrBadRequest("transaction_hash is empty"))
		return
	}
	rtv, err := srh.GetEventDB().GetErrorByTransactionHash(transactionHash)
	if err != nil {
		common.Respond(w, r, nil, common.NewErrInternal(err.Error()))
		return
	}
	common.Respond(w, r, rtv, nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/writemarkers writemarkers
// Gets list of write markers satisfying filter
//
// parameters:
//    + name: offset
//      description: offset
//      in: query
//      type: string
//    + name: limit
//      description: limit
//      in: query
//      type: string
//    + name: is_descending
//      description: is descending
//      in: query
//      type: string
//
// responses:
//  200: []WriteMarker
//  400:
//  500:
func (srh *StorageRestHandler) getWriteMarker(w http.ResponseWriter, r *http.Request) {
	var (
		offsetString       = r.URL.Query().Get("offset")
		limitString        = r.URL.Query().Get("limit")
		isDescendingString = r.URL.Query().Get("is_descending")
	)
	if offsetString == "" {
		offsetString = "0"
	}
	if limitString == "" {
		limitString = "10"
	}
	offset, err := strconv.Atoi(offsetString)
	if err != nil {
		common.Respond(w, r, nil, common.NewErrBadRequest("offset value was not valid: "+err.Error()))
		return
	}

	limit, err := strconv.Atoi(limitString)
	if err != nil {
		common.Respond(w, r, nil, common.NewErrBadRequest("limitString value was not valid: "+err.Error()))
		return
	}
	isDescending, err := strconv.ParseBool(isDescendingString)
	if err != nil {
		common.Respond(w, r, nil, common.NewErrBadRequest("is_descending value was not valid: "+err.Error()))
		return
	}

	rtv, err := srh.GetEventDB().GetWriteMarkers(offset, limit, isDescending)
	if err != nil {
		common.Respond(w, r, nil, common.NewErrInternal(err.Error()))
		return
	}
	common.Respond(w, r, rtv, nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/transactions transactions
// Gets filtered list of transaction information
//
// parameters:
//    + name: client_id
//      description: restrict to transactions sent by the specified client
//      in: query
//      type: string
//    + name: offset
//      description: offset
//      in: query
//      type: string
//    + name: limit
//      description: limit
//      in: query
//      type: string
//    + name: block_hash
//      description: restrict to transactions in indicated block
//      in: query
//      type: string
//
// responses:
//  200: []Transaction
//  400:
//  500:
func (srh *StorageRestHandler) getTransactionByFilter(w http.ResponseWriter, r *http.Request) {
	var (
		clientID     = r.URL.Query().Get("client_id")
		offsetString = r.URL.Query().Get("offset")
		limitString  = r.URL.Query().Get("limit")
		blockHash    = r.URL.Query().Get("block_hash")
	)
	if offsetString == "" {
		offsetString = "0"
	}
	if limitString == "" {
		limitString = "10"
	}
	offset, err := strconv.Atoi(offsetString)
	if err != nil {
		common.Respond(w, r, nil, common.NewErrBadRequest("offset value was not valid:"+err.Error()))
		return
	}

	limit, err := strconv.Atoi(limitString)
	if err != nil {
		common.Respond(w, r, nil, common.NewErrBadRequest("limitString value was not valid:"+err.Error()))
		return
	}

	if clientID != "" {
		rtv, err := srh.GetEventDB().GetTransactionByClientId(clientID, offset, limit)
		if err != nil {
			common.Respond(w, r, nil, common.NewErrInternal(err.Error()))
			return
		}
		common.Respond(w, r, rtv, nil)
		return
	}

	if blockHash != "" {
		rtv, err := srh.GetEventDB().GetTransactionByBlockHash(blockHash, offset, limit)
		if err != nil {
			common.Respond(w, r, nil, common.NewErrInternal(err.Error()))
			return
		}
		common.Respond(w, r, rtv, nil)
		return
	}

	common.Respond(w, r, nil, common.NewErrBadRequest("No filter selected"))

}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/transaction transaction
// Gets transaction information from transaction hash
//
// responses:
//  200: Transaction
//  500:
func (srh *StorageRestHandler) getTransactionByHash(w http.ResponseWriter, r *http.Request) {
	var transactionHash = r.URL.Query().Get("transaction_hash")
	if len(transactionHash) == 0 {
		err := common.NewErrBadRequest("cannot find valid transaction: transaction_hash is empty")
		common.Respond(w, r, nil, err)
		return
	}
	transaction, err := srh.GetEventDB().GetTransactionByHash(transactionHash)
	if err != nil {
		err := common.NewErrInternal("cannot get transaction: " + err.Error())
		common.Respond(w, r, nil, err)
		return
	}

	common.Respond(w, r, transaction, nil)
}

// getBlobbers swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/getblobbers getblobbers
// Gets list of all blobbers alive (e.g. excluding blobbers with zero capacity).
//
// responses:
//  200: StorageNodes
//  500:
func (srh *StorageRestHandler) getBlobbers(w http.ResponseWriter, r *http.Request) {
	blobbers, err := srh.GetEventDB().GetBlobbers()
	if err != nil || len(blobbers) == 0 {
		err := common.NewErrInternal("cannot get blobber list" + err.Error())
		common.Respond(w, r, nil, err)
		return
	}

	var sns storageNodesResponse
	for _, blobber := range blobbers {
		sn, err := blobberTableToStorageNode(blobber)
		if err != nil {
			err := common.NewErrInternal("parsing blobber" + blobber.BlobberID)
			common.Respond(w, r, nil, err)
			return
		}
		sns.Nodes = append(sns.Nodes, sn)
	}
	common.Respond(w, r, sns, nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/get_blobber_lat_long get_blobber_lat_long
// Gets list of latitude and longitude for all blobbers
//
// responses:
//  200: BlobberLatLong
//  500:
func (srh *StorageRestHandler) getBlobberGeoLocation(w http.ResponseWriter, r *http.Request) {
	blobbers, err := srh.GetEventDB().GetAllBlobberLatLong()
	if err != nil {
		err := common.NewErrInternal("cannot get blobber geolocation" + err.Error())
		common.Respond(w, r, nil, err)
		return
	}

	common.Respond(w, r, blobbers, nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/get_blobber_total_stakes get_blobber_total_stakes
// Gets total stake of all blobbers combined
//
// responses:
//  200: int64Map
//  500:
func (srh *StorageRestHandler) getBlobberTotalStakes(w http.ResponseWriter, r *http.Request) {
	blobbers, err := srh.GetEventDB().GetAllBlobberId()
	if err != nil {
		err := common.NewErrInternal("cannot get blobber list" + err.Error())
		common.Respond(w, r, nil, err)
		return
	}
	var total int64
	for _, blobber := range blobbers {
		var sp storageStakePool
		if err := sp.get(blobber, *srh); err != nil {
			err := common.NewErrInternal("cannot get stake pool" + err.Error())
			common.Respond(w, r, nil, err)
			return
		}
		total += int64(sp.Stake())
	}
	common.Respond(w, r, int64Map{
		"total": total,
	}, nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/get_blobber_count get_blobber_count
// Get count of blobber
//
// responses:
//  200: int64Map
//  400:
func (srh StorageRestHandler) getBlobberCount(w http.ResponseWriter, r *http.Request) {
	blobberCount, err := srh.GetEventDB().GetBlobberCount()
	if err != nil {
		err := common.NewErrInternal("getting blobber count:" + err.Error())
		common.Respond(w, r, nil, err)
		return
	}

	common.Respond(w, r, int64Map{
		"count": blobberCount,
	}, nil)
}

// swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/getBlobber getBlobber
// Get blobber information
//
// parameters:
//    + name: blobber_id
//      description: blobber for which to return information
//      required: true
//      in: query
//      type: string
//
// responses:
//  200: StorageNode
//  400:
//  500:
func (srh StorageRestHandler) getBlobber(w http.ResponseWriter, r *http.Request) {
	var blobberID = r.URL.Query().Get("blobber_id")
	if blobberID == "" {
		err := common.NewErrBadRequest("missing 'blobber_id' URL query parameter")
		common.Respond(w, r, nil, err)
		return
	}

	blobber, err := srh.GetEventDB().GetBlobber(blobberID)
	if err != nil {
		err := common.NewErrInternal("missing blobber" + blobberID)
		common.Respond(w, r, nil, err)
		return
	}

	sn, err := blobberTableToStorageNode(*blobber)
	if err != nil {
		err := common.NewErrInternal("parsing blobber" + blobberID)
		common.Respond(w, r, nil, err)
		return
	}
	common.Respond(w, r, sn, nil)
}
