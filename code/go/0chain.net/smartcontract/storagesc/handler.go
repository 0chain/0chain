package storagesc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"0chain.net/chaincore/currency"

	"go.uber.org/zap"

	"0chain.net/smartcontract/stakepool/spenum"

	"0chain.net/core/datastore"

	"0chain.net/smartcontract/stakepool"

	"0chain.net/smartcontract"
	"0chain.net/smartcontract/dbs/event"

	"0chain.net/core/logging"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/core/common"
	"0chain.net/core/util"
)

type storageNodesResponse struct {
	Nodes []storageNodeResponse
}

// StorageNode represents Blobber configurations.
type storageNodeResponse struct {
	StorageNode
	TotalStake int64 `json:"total_stake"`
}

func blobberTableToStorageNode(blobber event.Blobber) storageNodeResponse {
	return storageNodeResponse{
		StorageNode: StorageNode{
			ID:      blobber.BlobberID,
			BaseURL: blobber.BaseURL,
			Geolocation: StorageNodeGeolocation{
				Latitude:  blobber.Latitude,
				Longitude: blobber.Longitude,
			},
			Terms: Terms{
				ReadPrice:               currency.Coin(blobber.ReadPrice),
				WritePrice:              currency.Coin(blobber.WritePrice),
				MinLockDemand:           blobber.MinLockDemand,
				MaxOfferDuration:        time.Duration(blobber.MaxOfferDuration),
				ChallengeCompletionTime: time.Duration(blobber.ChallengeCompletionTime),
			},
			Capacity:        blobber.Capacity,
			Used:            blobber.Used,
			LastHealthCheck: common.Timestamp(blobber.LastHealthCheck),
			StakePoolSettings: stakepool.Settings{
				DelegateWallet:     blobber.DelegateWallet,
				MinStake:           currency.Coin(blobber.MinStake),
				MaxStake:           currency.Coin(blobber.MaxStake),
				MaxNumDelegates:    blobber.NumDelegates,
				ServiceChargeRatio: blobber.ServiceCharge,
			},
			Information: Info{
				Name:        blobber.Name,
				WebsiteUrl:  blobber.WebsiteUrl,
				LogoUrl:     blobber.LogoUrl,
				Description: blobber.Description,
			},
		},
		TotalStake: blobber.TotalStake,
	}
}

// Deprecated

// GetBlobberHandler returns Blobber object from its individual stored value.
func (ssc *StorageSmartContract) GetBlobberHandlerDepreciated(ctx context.Context,
	params url.Values, balances cstate.StateContextI) (
	resp interface{}, err error) {

	var blobberID = params.Get("blobber_id")
	if blobberID == "" {
		return nil, common.NewErrBadRequest("missing 'blobber_id' URL query parameter")
	}

	bl, err := ssc.getBlobber(blobberID, balances)
	if err != nil {
		return nil, smartcontract.NewErrNoResourceOrErrInternal(err, true, "can't get blobber")
	}

	return bl, nil
}

// GetBlobberHandler returns Blobber object from its individual stored value.
func (ssc *StorageSmartContract) GetBlobberHandler(
	ctx context.Context,
	params url.Values,
	balances cstate.StateContextI,
) (resp interface{}, err error) {
	var blobberID = params.Get("blobber_id")
	if blobberID == "" {
		return nil, common.NewErrBadRequest("missing 'blobber_id' URL query parameter")
	}

	if balances.GetEventDB() == nil {
		return ssc.GetBlobberHandlerDepreciated(ctx, params, balances)
	}

	blobber, err := balances.GetEventDB().GetBlobber(blobberID)
	if err != nil {
		return ssc.GetBlobberHandlerDepreciated(ctx, params, balances)
	}

	sn := blobberTableToStorageNode(*blobber)
	return sn, nil
}

// GetBlobberCountHandler returns Blobber count from its individual stored value.
func (ssc *StorageSmartContract) GetBlobberCountHandler(
	_ context.Context,
	_ url.Values,
	balances cstate.StateContextI,
) (resp interface{}, err error) {
	blobberCount, err := balances.GetEventDB().GetBlobberCount()
	if err != nil {
		return nil, fmt.Errorf("error while geting the blobber count")
	}
	return map[string]int64{
		"count": blobberCount,
	}, nil
}

// GetBlobberTotalStakesHandler returns blobber total stake
func (ssc *StorageSmartContract) GetBlobberTotalStakesHandler(
	_ context.Context,
	_ url.Values,
	balances cstate.StateContextI,
) (resp interface{}, err error) {
	if balances.GetEventDB() == nil {
		return nil, fmt.Errorf("Unable to connect to eventdb database")
	}
	blobbers, err := balances.GetEventDB().GetAllBlobberId()
	if err != nil {
		return nil, err
	}
	var total int64
	for _, blobber := range blobbers {
		sp, err := ssc.getStakePool(blobber, balances)
		if err != nil {
			return nil, err
		}
		total += int64(sp.stake())
	}
	return map[string]int64{
		"total": total,
	}, nil
}

// GetBlobberLatitudeLongitudeHandler returns blobber latitude and longitude
func (ssc *StorageSmartContract) GetBlobberLatitudeLongitudeHandler(
	_ context.Context,
	_ url.Values,
	balances cstate.StateContextI,
) (resp interface{}, err error) {
	if balances.GetEventDB() == nil {
		return nil, fmt.Errorf("unable to connect to eventdb database")
	}
	blobbers, err := balances.GetEventDB().GetAllBlobberLatLong()
	if err != nil {
		return nil, err
	}
	return blobbers, nil
}

// GetBlobbersHandler returns list of all blobbers alive (e.g. excluding
// blobbers with zero capacity).
func (ssc *StorageSmartContract) GetBlobbersHandler(
	ctx context.Context,
	params url.Values, balances cstate.StateContextI,
) (interface{}, error) {
	if balances.GetEventDB() == nil {
		return nil, common.NewErrInternal("events db is not initialised")
	}
	blobbers, err := balances.GetEventDB().GetBlobbers()
	if err != nil {
		return nil, err
	}

	var sns storageNodesResponse
	sns.Nodes = make([]storageNodeResponse, len(blobbers))
	for i, blobber := range blobbers {
		sn := blobberTableToStorageNode(blobber)
		sns.Nodes[i] = sn
	}
	return sns, nil
}

// GetAllocationBlobbersHandler returns list of all blobbers alive that match the allocation request.
func (ssc *StorageSmartContract) GetAllocationBlobbersHandler(
	ctx context.Context,
	params url.Values, balances cstate.StateContextI) (interface{}, error) {
	if balances.GetEventDB() == nil {
		return nil, errors.New("events db is not initialised")
	}

	var err error
	allocData := params.Get("allocation_data")
	var request newAllocationRequest
	if err := request.decode([]byte(allocData)); err != nil {
		return "", common.NewErrInternal("can't decode allocation request", err.Error())
	}

	blobberIDs, err := ssc.getBlobbersForRequest(request, balances)

	if err != nil {
		return "", err
	}

	return blobberIDs, nil
}

func (ssc *StorageSmartContract) getBlobbersForRequest(request newAllocationRequest, balances cstate.StateContextI) ([]string, error) {
	var sa = request.storageAllocation()
	var conf *Config
	var err error
	if conf, err = ssc.getConfig(balances, true); err != nil {
		return nil, fmt.Errorf("can't get config: %v", err)
	}

	var creationDate = time.Now()
	sa.TimeUnit = conf.TimeUnit // keep the initial time unit
	//if err = sa.validate(creationDate, conf); err != nil {
	//	return nil, fmt.Errorf("invalid request: %v", err)
	//}
	// number of blobbers required
	var numberOfBlobbers = sa.DataShards + sa.ParityShards
	if numberOfBlobbers > conf.MaxBlobbersPerAllocation {
		return nil, common.NewErrorf("allocation_creation_failed",
			"Too many blobbers selected, max available %d", conf.MaxBlobbersPerAllocation)
	}
	// size of allocation for a blobber
	var allocationSize = sa.bSize()
	dur := common.ToTime(sa.Expiration).Sub(creationDate)
	blobberIDs, err := balances.GetEventDB().GetBlobbersFromParams(event.AllocationQuery{
		MaxChallengeCompletionTime: request.MaxChallengeCompletionTime,
		MaxOfferDuration:           dur,
		ReadPriceRange: struct {
			Min int64
			Max int64
		}{
			Min: int64(request.ReadPriceRange.Min),
			Max: int64(request.ReadPriceRange.Max),
		},
		WritePriceRange: struct {
			Min int64
			Max int64
		}{
			Min: int64(request.WritePriceRange.Min),
			Max: int64(request.WritePriceRange.Max),
		},
		Size:              int(request.Size),
		AllocationSize:    allocationSize,
		PreferredBlobbers: request.Blobbers,
		NumberOfBlobbers:  numberOfBlobbers,
	})
	if err != nil {
		logging.Logger.Error("get_blobbers_for_request", zap.Error(err))
		return nil, errors.New("not enough blobbers to honor the allocation")
	}

	if err != nil || len(blobberIDs) < numberOfBlobbers {
		return nil, errors.New("not enough blobbers to honor the allocation")
	}
	return blobberIDs, nil
}

// GetFreeAllocationBlobbersHandler returns list of all blobbers alive that match the free allocation request.
func (ssc *StorageSmartContract) GetFreeAllocationBlobbersHandler(ctx context.Context, params url.Values,
	balances cstate.StateContextI) (interface{}, error) {
	var err error
	allocData := params.Get("free_allocation_data")
	var inputObj freeStorageAllocationInput
	if err := inputObj.decode([]byte(allocData)); err != nil {
		return "", common.NewErrInternal("can't decode allocation request", err.Error())
	}

	var marker freeStorageMarker
	if err := marker.decode([]byte(inputObj.Marker)); err != nil {
		return "", common.NewErrorf("free_allocation_failed",
			"unmarshal request: %v", err)
	}

	var conf *Config
	if conf, err = ssc.getConfig(balances, true); err != nil {
		return "", common.NewErrorf("free_allocation_failed",
			"can't get config: %v", err)
	}

	request := newAllocationRequest{
		DataShards:                 conf.FreeAllocationSettings.DataShards,
		ParityShards:               conf.FreeAllocationSettings.ParityShards,
		Size:                       conf.FreeAllocationSettings.Size,
		Expiration:                 common.Timestamp(time.Now().Add(conf.FreeAllocationSettings.Duration).Unix()),
		Owner:                      marker.Recipient,
		OwnerPublicKey:             inputObj.RecipientPublicKey,
		ReadPriceRange:             conf.FreeAllocationSettings.ReadPriceRange,
		WritePriceRange:            conf.FreeAllocationSettings.WritePriceRange,
		MaxChallengeCompletionTime: conf.FreeAllocationSettings.MaxChallengeCompletionTime,
		Blobbers:                   inputObj.Blobbers,
	}

	return ssc.getBlobbersForRequest(request, balances)

}

func (ssc *StorageSmartContract) GetBlobberIdsByUrlsHandler(
	ctx context.Context,
	params url.Values,
	balances cstate.StateContextI,
) (interface{}, error) {
	urlsStr := params.Get("blobber_urls")
	if len(urlsStr) == 0 {
		return nil, errors.New("blobber urls list is empty")
	}
	if balances.GetEventDB() == nil {
		return nil, errors.New("no event database found")
	}

	var urls []string
	err := json.Unmarshal([]byte(urlsStr), &urls)
	if err != nil {
		return nil, errors.New("blobber urls list is malformed")
	}

	if len(urls) == 0 {
		return make([]string, 0), nil
	}

	ids, err := balances.GetEventDB().GetBlobberIdsFromUrls(urls)
	return ids, err
}

func (ssc *StorageSmartContract) GetTransactionByHashHandler(
	ctx context.Context,
	params url.Values,
	balances cstate.StateContextI,
) (interface{}, error) {
	var transactionHash = params.Get("transaction_hash")
	if len(transactionHash) == 0 {
		return nil, errors.New("cannot find valid transaction: transaction_hash is empty")
	}
	if balances.GetEventDB() == nil {
		return nil, errors.New("no event database found")
	}
	transaction, err := balances.GetEventDB().GetTransactionByHash(transactionHash)
	return transaction, err
}

func (msc *StorageSmartContract) GetTransactionByFilterHandler(
	ctx context.Context,
	params url.Values,
	balances cstate.StateContextI,
) (interface{}, error) {
	var (
		clientID     = params.Get("client_id")
		offsetString = params.Get("offset")
		limitString  = params.Get("limit")
		blockHash    = params.Get("block_hash")
	)
	if offsetString == "" {
		offsetString = "0"
	}
	if limitString == "" {
		limitString = "10"
	}
	offset, err := strconv.Atoi(offsetString)
	if err != nil {
		return nil, errors.New("offset value was not valid")
	}

	limit, err := strconv.Atoi(limitString)
	if err != nil {
		return nil, errors.New("limitString value was not valid")
	}

	if balances.GetEventDB() == nil {
		return nil, errors.New("no event database found")
	}

	if clientID != "" {
		return balances.GetEventDB().GetTransactionByClientId(clientID, offset, limit)
	}
	if blockHash != "" {
		return balances.GetEventDB().GetTransactionByBlockHash(blockHash, offset, limit)
	}
	return nil, errors.New("no filter selected")
}

func (msc *StorageSmartContract) GetWriteMarkerHandler(
	ctx context.Context,
	params url.Values,
	balances cstate.StateContextI,
) (interface{}, error) {
	var (
		offsetString       = params.Get("offset")
		limitString        = params.Get("limit")
		isDescendingString = params.Get("is_descending")
	)
	if offsetString == "" {
		offsetString = "0"
	}
	if limitString == "" {
		limitString = "10"
	}
	offset, err := strconv.Atoi(offsetString)
	if err != nil {
		return nil, errors.New("offset value was not valid")
	}

	limit, err := strconv.Atoi(limitString)
	if err != nil {
		return nil, errors.New("limitString value was not valid")
	}
	isDescending, err := strconv.ParseBool(isDescendingString)
	if err != nil {
		return nil, errors.New("is_descending value was not valid")
	}
	if balances.GetEventDB() == nil {
		return nil, errors.New("no event database found")
	}
	return balances.GetEventDB().GetWriteMarkers(offset, limit, isDescending)
}

func (msc *StorageSmartContract) GetErrors(
	ctx context.Context,
	params url.Values,
	balances cstate.StateContextI,
) (interface{}, error) {
	transactionHash := params.Get("transaction_hash")
	if len(transactionHash) == 0 {
		return nil, fmt.Errorf("cannot find valid transaction_hash: %v", transactionHash)
	}
	if balances.GetEventDB() == nil {
		return nil, errors.New("no event database found")
	}
	transaction, err := balances.GetEventDB().GetErrorByTransactionHash(transactionHash)
	return &transaction, err
}

func (ssc *StorageSmartContract) GetAllocationsHandlerDeprecated(ctx context.Context,
	params url.Values, balances cstate.StateContextI) (interface{}, error) {

	logging.Logger.Info("GetAllocationsHandler",
		zap.Bool("is event db present", balances.GetEventDB() != nil))

	clientID := params.Get("client")
	allocations, err := ssc.getAllocationsList(clientID, balances)
	if err != nil {
		return nil, common.NewErrInternal("can't get allocation list", err.Error())
	}
	result := make([]*StorageAllocationBlobbers, len(allocations.List))
	for _, allocationID := range allocations.List {
		allocationObj := &StorageAllocationBlobbers{}
		allocationObj.ID = allocationID

		err := balances.GetTrieNode(allocationObj.GetKey(ssc.ID), allocationObj)
		switch err {
		case nil:
			err = allocationObj.getBlobbers(ssc, balances)
			if err != nil {
				return nil, smartcontract.NewErrNoResourceOrErrInternal(err, true, cantGetBlobber)
			}
			result = append(result, allocationObj)
		case util.ErrValueNotPresent:
			continue
		default:
			msg := fmt.Sprintf("can't decode allocation with id '%s'", allocationID)
			return nil, common.NewErrInternal(msg, err.Error())
		}
	}
	return result, nil
}

func (ssc *StorageSmartContract) GetAllocationsHandler(ctx context.Context,
	params url.Values, balances cstate.StateContextI) (interface{}, error) {

	clientID := params.Get("client")

	if balances.GetEventDB() == nil {
		return ssc.GetAllocationsHandlerDeprecated(ctx, params, balances)
	}

	allocations, err := getClientAllocationsFromDb(clientID, balances.GetEventDB())
	if err != nil {
		return nil, smartcontract.NewErrNoResourceOrErrInternal(err, true, "can't get allocations")
	}

	return allocations, nil
}

func (ssc *StorageSmartContract) GetActiveAllocationsCountHandler(ctx context.Context,
	params url.Values, balances cstate.StateContextI) (interface{}, error) {

	if balances.GetEventDB() == nil {
		return nil, common.NewErrNoResource("db is not initialized")
	}
	count, err := balances.GetEventDB().GetActiveAllocationsCount()
	if err != nil {
		return nil, smartcontract.NewErrNoResourceOrErrInternal(err, true,
			"can't get active allocations count")
	}

	response := struct {
		ActiveAllocationsCount int64 `json:"active_allocations_count"`
	}{count}

	return response, nil
}

func (ssc *StorageSmartContract) GetActiveAllocsBlobberCountHandler(ctx context.Context,
	params url.Values, balances cstate.StateContextI) (interface{}, error) {

	if balances.GetEventDB() == nil {
		return nil, common.NewErrNoResource("db is not initialized")
	}

	count, err := balances.GetEventDB().GetActiveAllocsBlobberCount()
	if err != nil {
		return nil, smartcontract.NewErrNoResourceOrErrInternal(err, true,
			"can't get blobber allocations count")
	}

	response := struct {
		BlobberAllocationsCount int64 `json:"blobber_allocations_count"`
	}{count}

	return response, nil
}

func (ssc *StorageSmartContract) GetAllocationMinLockHandler(ctx context.Context,
	params url.Values, balances cstate.StateContextI) (interface{}, error) {

	var err error
	creationDate := time.Now()

	allocData := params.Get("allocation_data")
	var req newAllocationRequest
	if err = req.decode([]byte(allocData)); err != nil {
		return "", common.NewErrInternal("can't decode allocation request", err.Error())
	}

	blobbers, err := ssc.getBlobbersForRequest(req, balances)
	if err != nil {
		return "", common.NewErrInternal("error selecting blobbers", err.Error())
	}
	sa := req.storageAllocation()
	var gbSize = sizeInGB(sa.bSize())
	var minLockDemand currency.Coin

	ids := append(req.Blobbers, blobbers...)
	uniqueMap := make(map[string]struct{})
	for _, id := range ids {
		uniqueMap[id] = struct{}{}
	}
	unique := make([]string, 0, len(ids))
	for id := range uniqueMap {
		unique = append(unique, id)
	}
	if len(unique) > req.ParityShards+req.DataShards {
		unique = unique[:req.ParityShards+req.DataShards]
	}

	nodes := ssc.getBlobbers(unique, balances)
	for _, b := range nodes.Nodes {
		minLockDemand += b.Terms.minLockDemand(gbSize,
			sa.restDurationInTimeUnits(common.Timestamp(creationDate.Unix())))
	}

	var response = map[string]interface{}{
		"min_lock_demand": minLockDemand,
	}

	return response, nil
}

const (
	cantGetAllocation = "can't get allocation"
	cantGetBlobber    = "can't get blobber"
)

func (ssc *StorageSmartContract) AllocationStatsHandlerDeprecated(ctx context.Context, params url.Values, balances cstate.StateContextI) (interface{}, error) {
	logging.Logger.Info("AllocationStatsHandler",
		zap.Bool("is event db present", balances.GetEventDB() != nil))
	allocationID := params.Get("allocation")
	allocationObj := &StorageAllocationBlobbers{}
	allocationObj.ID = allocationID

	err := balances.GetTrieNode(allocationObj.GetKey(ssc.ID), allocationObj)
	if err != nil {
		return nil, smartcontract.NewErrNoResourceOrErrInternal(err, true, cantGetAllocation)
	}

	err = allocationObj.getBlobbers(ssc, balances)
	if err != nil {
		return nil, smartcontract.NewErrNoResourceOrErrInternal(err, true, cantGetBlobber)
	}

	return allocationObj, nil
}

func (ssc *StorageSmartContract) AllocationStatsHandler(ctx context.Context, params url.Values, balances cstate.StateContextI) (interface{}, error) {
	allocationID := params.Get("allocation")

	if balances.GetEventDB() == nil {
		return ssc.AllocationStatsHandlerDeprecated(ctx, params, balances)
	}
	allocation, err := getStorageAllocationFromDb(allocationID, balances.GetEventDB())
	if err != nil {
		return nil, smartcontract.NewErrNoResourceOrErrInternal(err, true, cantGetAllocation)
	}

	return allocation, nil
}

func (ssc *StorageSmartContract) LatestReadMarkerHandler(ctx context.Context,
	params url.Values, balances cstate.StateContextI) (
	resp interface{}, err error) {

	var (
		clientID  = params.Get("client")
		blobberID = params.Get("blobber")

		commitRead = &ReadConnection{}
	)

	commitRead.ReadMarker = &ReadMarker{
		BlobberID: blobberID,
		ClientID:  clientID,
	}

	err = balances.GetTrieNode(commitRead.GetKey(ssc.ID), commitRead)
	switch err {
	case nil:
		return commitRead.ReadMarker, nil // ok
	case util.ErrValueNotPresent:
		return make(map[string]string), nil
	default:
		return nil, common.NewErrInternal("can't get read marker", err.Error())
	}
}

func (ssc *StorageSmartContract) GetReadMarkersHandler(ctx context.Context,
	params url.Values, balances cstate.StateContextI) (
	resp interface{}, err error) {

	var (
		allocationID = params.Get("allocation_id")
		authTicket   = params.Get("auth_ticket")
		offsetString = params.Get("offset")
		limitString  = params.Get("limit")
		sortString   = params.Get("sort")
		limit        = 0
		offset       = 0
		isDescending = false
	)

	if balances.GetEventDB() == nil {
		return nil, common.NewErrNoResource("db not initialized")
	}

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
			return nil, errors.New("offset is invalid")
		}
		offset = o
	}

	if limitString != "" {
		l, err := strconv.Atoi(limitString)
		if err != nil {
			return nil, errors.New("limit is invalid")
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
			return nil, errors.New("sort value is invalid")
		}
	}

	readMarkers, err := balances.GetEventDB().GetReadMarkersFromQueryPaginated(query, offset, limit, isDescending)
	if err != nil {
		return nil, common.NewErrInternal("can't get read markers", err.Error())
	}

	return readMarkers, nil

}

func (ssc *StorageSmartContract) GetReadMarkersCount(ctx context.Context,
	params url.Values, balances cstate.StateContextI) (
	resp interface{}, err error) {

	var (
		allocationID = params.Get("allocation_id")
	)

	if allocationID == "" {
		return nil, common.NewErrInternal("Expecting params: allocation_id")
	}

	if balances.GetEventDB() == nil {
		return nil, common.NewErrNoResource("db not initialized")
	}

	query := new(event.ReadMarker)
	if allocationID != "" {
		query.AllocationID = allocationID
	}

	count, err := balances.GetEventDB().CountReadMarkersFromQuery(query)
	if err != nil {
		return nil, common.NewErrInternal("can't count read markers", err.Error())
	}

	return struct {
		ReadMarkersCount int64 `json:"read_markers_count"`
	}{
		ReadMarkersCount: count,
	}, nil

}

func (ssc *StorageSmartContract) GetWriteMarkersHandler(ctx context.Context,
	params url.Values, balances cstate.StateContextI) (
	resp interface{}, err error) {

	var (
		allocationID = params.Get("allocation_id")
	)

	if allocationID == "" {
		return nil, common.NewErrInternal("allocation id is empty")
	}

	if balances.GetEventDB() == nil {
		return nil, common.NewErrNoResource("db not initialized")
	}

	filename := params.Get("filename")

	if filename == "" {
		writeMarkers, err := balances.GetEventDB().GetWriteMarkersForAllocationID(allocationID)
		if err != nil {
			return nil, common.NewErrInternal("can't get write markers", err.Error())
		}

		return writeMarkers, nil
	} else {
		writeMarkers, err := balances.GetEventDB().GetWriteMarkersForAllocationFile(allocationID, filename)
		if err != nil {
			return nil, common.NewErrInternal("can't get write markers for file", err.Error())
		}

		return writeMarkers, nil
	}
}

func (ssc *StorageSmartContract) GetWrittenAmountHandler(
	ctx context.Context,
	params url.Values,
	balances cstate.StateContextI,
) (interface{}, error) {
	if balances.GetEventDB() == nil {
		return nil, common.NewErrNoResource("db not initialized")
	}
	blockNumberString := params.Get("block_number")
	allocationIDString := params.Get("allocation_id")
	if blockNumberString == "" {
		return nil, common.NewErrInternal("block_number is empty")
	}
	blockNumber, err := strconv.Atoi(blockNumberString)
	if err != nil {
		return nil, common.NewErrInternal("block_number is not valid")
	}

	total, err := balances.GetEventDB().GetAllocationWrittenSizeInLastNBlocks(int64(blockNumber), allocationIDString)
	return map[string]int64{
		"total": total,
	}, err
}

func (ssc *StorageSmartContract) GetReadAmountHandler(
	ctx context.Context,
	params url.Values,
	balances cstate.StateContextI,
) (interface{}, error) {
	if balances.GetEventDB() == nil {
		return nil, common.NewErrNoResource("db not initialized")
	}
	blockNumberString := params.Get("block_number")
	allocationIDString := params.Get("allocation_id")
	if blockNumberString == "" {
		return nil, common.NewErrInternal("block_number is empty")
	}
	blockNumber, err := strconv.Atoi(blockNumberString)
	if err != nil {
		return nil, common.NewErrInternal("block_number is not valid")
	}

	total, err := balances.GetEventDB().GetDataReadFromAllocationForLastNBlocks(int64(blockNumber), allocationIDString)
	return map[string]int64{
		"total": total,
	}, err
}

func (ssc *StorageSmartContract) GetWriteMarkerCountHandler(
	ctx context.Context,
	params url.Values,
	balances cstate.StateContextI,
) (interface{}, error) {
	if balances.GetEventDB() == nil {
		return nil, common.NewErrNoResource("db not initialized")
	}
	allocationID := params.Get("allocation_id")
	if allocationID == "" {
		return nil, common.NewErrInternal("allocation_id is empty")
	}

	total, err := balances.GetEventDB().GetWriteMarkerCount(allocationID)
	return map[string]int64{
		"count": total,
	}, err
}

func (ssc *StorageSmartContract) GetValidatorHandler(ctx context.Context,
	params url.Values, balances cstate.StateContextI) (
	resp interface{}, err error) {

	var (
		validatorID = params.Get("validator_id")
	)

	if validatorID == "" {
		return nil, common.NewErrInternal("validator id is empty")
	}

	if balances.GetEventDB() == nil {
		return nil, common.NewErrNoResource("Event db not initialized")
	}

	validator, err := balances.GetEventDB().GetValidatorByValidatorID(validatorID)
	if err != nil {
		return nil, common.NewErrInternal("can't get validator", err.Error())
	}

	return validator, nil

}

type StorageChallengeResponse struct {
	*StorageChallenge `json:",inline"`
	Validators        []*ValidationNode `json:"validators"`
	Seed              int64             `json:"seed"`
	AllocationRoot    string            `json:"allocation_root"`
}

type ChallengesResponse struct {
	BlobberID  string                      `json:"blobber_id"`
	Challenges []*StorageChallengeResponse `json:"challenges"`
}

func (ssc *StorageSmartContract) OpenChallengeHandler(ctx context.Context, params url.Values, balances cstate.StateContextI) (interface{}, error) {
	blobberID := params.Get("blobber")

	blobber, err := balances.GetEventDB().GetBlobber(blobberID)
	if err != nil {
		return "", smartcontract.NewErrNoResourceOrErrInternal(err, true, "can't find blobber")
	}

	challenges, err := getOpenChallengesForBlobber(blobberID, common.Timestamp(blobber.ChallengeCompletionTime), balances)
	if err != nil {
		return "", smartcontract.NewErrNoResourceOrErrInternal(err, true, "can't find challenges")
	}
	return ChallengesResponse{
		BlobberID:  blobberID,
		Challenges: challenges,
	}, nil
}

func (ssc *StorageSmartContract) GetChallengeHandler(ctx context.Context, params url.Values, balances cstate.StateContextI) (retVal interface{}, retErr error) {

	blobberID := params.Get("blobber")

	challengeID := params.Get("challenge")
	challenge, err := getChallengeForBlobber(blobberID, challengeID, balances)
	if err != nil {
		return "", smartcontract.NewErrNoResourceOrErrInternal(err, true, "can't get challenge")
	}

	return challenge, nil
}

// statistic for all locked tokens of a stake pool
func (ssc *StorageSmartContract) getStakePoolStatHandler(
	ctx context.Context,
	params url.Values,
	balances cstate.StateContextI,
) (interface{}, error) {
	blobberID := datastore.Key(params.Get("blobber_id"))
	if balances.GetEventDB() == nil {
		return nil, errors.New("no event database found")
	}

	blobber, err := balances.GetEventDB().GetBlobber(blobberID)
	if err != nil {
		return nil, errors.New("blobber not found in event database")
	}

	delegatePools, err := balances.GetEventDB().GetDelegatePools(blobberID, int(spenum.Blobber))
	if err != nil {
		return "", common.NewErrInternal("can't find user stake pool", err.Error())
	}

	return spStats(*blobber, delegatePools), nil
}

func spStats(
	blobber event.Blobber,
	delegatePools []event.DelegatePool,
) *stakePoolStat {
	stat := new(stakePoolStat)
	stat.ID = blobber.BlobberID
	stat.UnstakeTotal = currency.Coin(blobber.UnstakeTotal)
	stat.Capacity = blobber.Capacity
	stat.WritePrice = currency.Coin(blobber.WritePrice)
	stat.OffersTotal = currency.Coin(blobber.OffersTotal)
	stat.Delegate = make([]delegatePoolStat, 0, len(delegatePools))
	stat.Settings = stakepool.Settings{
		DelegateWallet:     blobber.DelegateWallet,
		MinStake:           currency.Coin(blobber.MinStake),
		MaxStake:           currency.Coin(blobber.MaxStake),
		MaxNumDelegates:    blobber.NumDelegates,
		ServiceChargeRatio: blobber.ServiceCharge,
	}
	stat.Rewards = currency.Coin(blobber.Reward)
	for _, dp := range delegatePools {
		dpStats := delegatePoolStat{
			ID:           dp.PoolID,
			Balance:      currency.Coin(dp.Balance),
			DelegateID:   dp.DelegateID,
			Rewards:      currency.Coin(dp.Reward),
			Status:       spenum.PoolStatus(dp.Status).String(),
			TotalReward:  currency.Coin(dp.TotalReward),
			TotalPenalty: currency.Coin(dp.TotalPenalty),
			RoundCreated: dp.RoundCreated,
		}
		stat.Balance += dpStats.Balance
		stat.Delegate = append(stat.Delegate, dpStats)
	}
	return stat
}

type userPoolStat struct {
	Pools map[datastore.Key][]*delegatePoolStat `json:"pools"`
}

func (ssc *StorageSmartContract) getUserStakePoolStatHandler(
	ctx context.Context,
	params url.Values,
	balances cstate.StateContextI,
) (resp interface{}, err error) {
	clientID := datastore.Key(params.Get("client_id"))

	if balances.GetEventDB() == nil {
		return nil, errors.New("no event database found")
	}

	pools, err := balances.GetEventDB().GetUserDelegatePools(clientID, int(spenum.Blobber))
	if err != nil {
		return nil, errors.New("blobber not found in event database")
	}

	var ups = new(userPoolStat)
	ups.Pools = make(map[datastore.Key][]*delegatePoolStat)
	for _, pool := range pools {
		var dps = delegatePoolStat{
			ID:           pool.PoolID,
			Balance:      currency.Coin(pool.Balance),
			DelegateID:   pool.DelegateID,
			Rewards:      currency.Coin(pool.Reward),
			TotalPenalty: currency.Coin(pool.TotalPenalty),
			TotalReward:  currency.Coin(pool.TotalReward),
			Status:       spenum.PoolStatus(pool.Status).String(),
			RoundCreated: pool.RoundCreated,
		}
		ups.Pools[pool.ProviderID] = append(ups.Pools[pool.ProviderID], &dps)
	}

	return ups, nil
}

func (ssc *StorageSmartContract) GetBlockByHashHandler(_ context.Context, params url.Values, balances cstate.StateContextI) (interface{}, error) {
	hash := params.Get("block_hash")
	if len(hash) == 0 {
		return nil, fmt.Errorf("cannot find valid block hash: %v", hash)
	}
	if balances.GetEventDB() == nil {
		return nil, errors.New("no event database found")
	}
	block, err := balances.GetEventDB().GetBlocksByHash(hash)
	return &block, err
}

func (ssc *StorageSmartContract) GetBlocksHandler(_ context.Context, params url.Values, balances cstate.StateContextI) (interface{}, error) {
	if balances.GetEventDB() == nil {
		return nil, errors.New("no event database found")
	}
	block, err := balances.GetEventDB().GetBlocks()
	return &block, err
}

func (ssc *StorageSmartContract) GetTotalData(_ context.Context, balances cstate.StateContextI) (int64, error) {
	return 0, fmt.Errorf("not implemented yet")
}

func (ssc *StorageSmartContract) GetCollectedRewardHandler(ctx context.Context, params url.Values, balances cstate.StateContextI) (resp interface{}, err error) {
	if balances.GetEventDB() == nil {
		return 0, common.NewErrNoResource("db not initialized")
	}

	var (
		startBlock, _ = strconv.Atoi(params.Get("start_block"))
		endBlock, _   = strconv.Atoi(params.Get("end_block"))
		clientID      = params.Get("client_id")
	)

	query := event.RewardQuery{
		StartBlock: startBlock,
		EndBlock:   endBlock,
		ClientID:   clientID,
	}

	collectedReward, err := balances.GetEventDB().GetRewardClaimedTotal(query)
	if err != nil {
		return 0, common.NewErrInternal("can't get rewards claimed", err.Error())
	}

	return map[string]int64{
		"collected_reward": collectedReward,
	}, nil
}
