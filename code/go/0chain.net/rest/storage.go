package rest

import (
	"net/http"
	"strconv"

	"0chain.net/smartcontract/dbs/event"

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
}

// getReadMarkers swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/readmarkers readmarkers
// Gets read markers according to a filter
//
// parameters:
//    + name: allocation_id
//      description: filter read markers by this allocation
//    + name: auth_ticket
//      description: filter in only read markers using auth thicket
//    + name: offset
//      description: offset
//    + name: limit
//      description: limit
//    + name: sort
//      description: desc or asc
//
// responses:
//  200: body:[]event.ReadMarker
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
			common.Respond(w, r, "", common.NewErrBadRequest("offset is invalid: "+err.Error()))
			return
		}
		offset = o
	}

	if limitString != "" {
		l, err := strconv.Atoi(limitString)
		if err != nil {
			common.Respond(w, r, "", common.NewErrBadRequest("limit is invalid: "+err.Error()))
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
			common.Respond(w, r, "", common.NewErrBadRequest("sort is invalid: "+sortString))
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

// getErrors swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/allocation allocation
// Gets latest read marker for a client and blobber
//
// parameters:
//    + name: client
//      description: client
//    + name: blobber
//      description: blobber
//
// responses:
//  200: body:[]event.Error
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

// getErrors swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/allocation allocation
// Gets allocation object
//
// parameters:
//    + name: transaction_hash
//      description: offset
//      required: true
//
// responses:
//  200: body:[]event.Error
//  400:
//  500:
func (srh *StorageRestHandler) getAllocationStats(w http.ResponseWriter, r *http.Request) {
	allocationID := r.URL.Query().Get("allocation")
	allocationObj := &storagesc.StorageAllocation{}
	allocationObj.ID = allocationID

	err := srh.GetTrieNode(allocationObj.GetKey(storagesc.ADDRESS), allocationObj)
	if err != nil {
		common.Respond(w, r, "", smartcontract.NewErrNoResourceOrErrInternal(err, true, "can't get allocation"))
		return
	}
	common.Respond(w, r, allocationObj, nil)
}

// getErrors swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/errors errors
// Gets errors returned by indicated transaction
//
// parameters:
//    + name: transaction_hash
//      description: offset
//      required: true
//
// responses:
//  200: body:[]event.Error
//  400:
//  500:
func (srh *StorageRestHandler) getErrors(w http.ResponseWriter, r *http.Request) {
	transactionHash := r.URL.Query().Get("transaction_hash")
	if len(transactionHash) == 0 {
		common.Respond(w, r, "", common.NewErrBadRequest("transaction_hash is empty"))
		return
	}
	rtv, err := srh.GetEventDB().GetErrorByTransactionHash(transactionHash)
	if err != nil {
		common.Respond(w, r, "", common.NewErrInternal(err.Error()))
		return
	}
	common.Respond(w, r, rtv, nil)
}

// GetWriteMarker swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/writemarkers writemarkers
// Gets list of write markers satisfying filter
//
// parameters:
//    + name: offset
//      description: offset
//    + name: limit
//      description: limit
//    + name: is_descending
//      description: is descending
//
// responses:
//  200: body:[]event.WriteMarker
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
		common.Respond(w, r, "", common.NewErrBadRequest("offset value was not valid: "+err.Error()))
		return
	}

	limit, err := strconv.Atoi(limitString)
	if err != nil {
		common.Respond(w, r, "", common.NewErrBadRequest("limitString value was not valid: "+err.Error()))
		return
	}
	isDescending, err := strconv.ParseBool(isDescendingString)
	if err != nil {
		common.Respond(w, r, "", common.NewErrBadRequest("is_descending value was not valid: "+err.Error()))
		return
	}

	rtv, err := srh.GetEventDB().GetWriteMarkers(offset, limit, isDescending)
	if err != nil {
		common.Respond(w, r, "", common.NewErrInternal(err.Error()))
		return
	}
	common.Respond(w, r, rtv, nil)
}

// GetTransactionByFilter swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/transactions transactions
// Gets filtered list of transaction information
//
// parameters:
//    + name: client_id
//      description: restrict to transactions sent by the specified client
//    + name: offset
//      description: offset
//    + name: limit
//      description: limit
//    + name: block_hash
//      description: restrict to transactions in indicated block
//
// responses:
//  200: body:[]event.Transaction
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
		common.Respond(w, r, "", common.NewErrBadRequest("offset value was not valid:"+err.Error()))
		return
	}

	limit, err := strconv.Atoi(limitString)
	if err != nil {
		common.Respond(w, r, "", common.NewErrBadRequest("limitString value was not valid:"+err.Error()))
		return
	}

	if clientID != "" {
		rtv, err := srh.GetEventDB().GetTransactionByClientId(clientID, offset, limit)
		if err != nil {
			common.Respond(w, r, "", common.NewErrInternal(err.Error()))
			return
		}
		common.Respond(w, r, rtv, nil)
		return
	}

	if blockHash != "" {
		rtv, err := srh.GetEventDB().GetTransactionByBlockHash(blockHash, offset, limit)
		if err != nil {
			common.Respond(w, r, "", common.NewErrInternal(err.Error()))
			return
		}
		common.Respond(w, r, rtv, nil)
		return
	}

	common.Respond(w, r, "", common.NewErrBadRequest("No filter selected"))

}

// getBlobbers swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/transaction transaction
// Gets transaction information from transaction hash
//
// responses:
//  200: body:event.Transaction
//  500:
func (srh *StorageRestHandler) getTransactionByHash(w http.ResponseWriter, r *http.Request) {
	var transactionHash = r.URL.Query().Get("transaction_hash")
	if len(transactionHash) == 0 {
		err := common.NewErrBadRequest("cannot find valid transaction: transaction_hash is empty")
		common.Respond(w, r, "", err)
		return
	}
	transaction, err := srh.GetEventDB().GetTransactionByHash(transactionHash)
	if err != nil {
		err := common.NewErrInternal("cannot get transaction: " + err.Error())
		common.Respond(w, r, "", err)
		return
	}

	common.Respond(w, r, transaction, nil)
}

// getBlobbers swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/getblobbers getblobbers
// Gets list of all blobbers alive (e.g. excluding blobbers with zero capacity).
//
// responses:
//  200: body:storagesc.StorageNodes
//  500:
func (srh *StorageRestHandler) getBlobbers(w http.ResponseWriter, r *http.Request) {
	blobbers, err := srh.GetEventDB().GetBlobbers()
	if err != nil || len(blobbers) == 0 {
		err := common.NewErrInternal("cannot get blobber list" + err.Error())
		common.Respond(w, r, "", err)
		return
	}

	var sns storagesc.StorageNodes
	for _, blobber := range blobbers {
		sn, err := blobberTableToStorageNode(blobber)
		if err != nil {
			err := common.NewErrInternal("parsing blobber" + blobber.BlobberID)
			common.Respond(w, r, "", err)
			return
		}
		ssn := storagesc.StorageNode(sn)
		sns.Nodes.Add(&ssn)
	}
	common.Respond(w, r, sns, nil)
}

// getBlobberGeoLocation swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/get_blobber_lat_long get_blobber_lat_long
// Gets list of latitude and longitude for all blobbers
//
// responses:
//  200: body:event.BlobberLatLong
//  500:
func (srh *StorageRestHandler) getBlobberGeoLocation(w http.ResponseWriter, r *http.Request) {
	blobbers, err := srh.GetEventDB().GetAllBlobberLatLong()
	if err != nil {
		err := common.NewErrInternal("cannot get blobber geolocation" + err.Error())
		common.Respond(w, r, "", err)
		return
	}

	common.Respond(w, r, blobbers, nil)
}

// GetBlobberTotalStakes swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/get_blobber_total_stakes get_blobber_total_stakes
// Gets total stake of all blobbers combined
//
// responses:
//  200: intMap
//  500:
func (srh *StorageRestHandler) getBlobberTotalStakes(w http.ResponseWriter, r *http.Request) {
	blobbers, err := srh.GetEventDB().GetAllBlobberId()
	if err != nil {
		err := common.NewErrInternal("cannot get blobber list" + err.Error())
		common.Respond(w, r, "", err)
		return
	}
	var total int64
	for _, blobber := range blobbers {
		var sp storageStakePool
		if err := sp.get(blobber, *srh); err != nil {
			err := common.NewErrInternal("cannot get stake pool" + err.Error())
			common.Respond(w, r, "", err)
			return
		}
		total += int64(sp.Stake())
	}
	common.Respond(w, r, int64Map{
		"total": total,
	}, nil)
}

// GetBlobberCount swagger:route GET /v1/screst/6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7/get_blobber_count get_blobber_count
// Get count of blobber
//
// responses:
//  200: intMap
//  400:
func (srh StorageRestHandler) getBlobberCount(w http.ResponseWriter, r *http.Request) {
	blobberCount, err := srh.GetEventDB().GetBlobberCount()
	if err != nil {
		err := common.NewErrInternal("getting blobber count:" + err.Error())
		common.Respond(w, r, "", err)
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
//
// responses:
//  200: StorageNode
//  400:
//  500:
func (srh StorageRestHandler) getBlobber(w http.ResponseWriter, r *http.Request) {
	var blobberID = r.URL.Query().Get("blobber_id")
	if blobberID == "" {
		err := common.NewErrBadRequest("missing 'blobber_id' URL query parameter")
		common.Respond(w, r, "", err)
		return
	}

	blobber, err := srh.GetEventDB().GetBlobber(blobberID)
	if err != nil {
		err := common.NewErrInternal("missing blobber" + blobberID)
		common.Respond(w, r, "", err)
		return
	}

	sn, err := blobberTableToStorageNode(*blobber)
	if err != nil {
		err := common.NewErrInternal("parsing blobber" + blobberID)
		common.Respond(w, r, "", err)
		return
	}
	common.Respond(w, r, sn, nil)
}
