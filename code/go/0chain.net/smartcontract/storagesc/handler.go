package storagesc

import (
	"0chain.net/smartcontract/provider"
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"go.uber.org/zap"

	"0chain.net/smartcontract/stakepool/spenum"

	"0chain.net/core/datastore"

	"0chain.net/smartcontract/stakepool"

	"0chain.net/smartcontract"
	"0chain.net/smartcontract/dbs/event"

	"0chain.net/core/logging"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
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

func blobberTableToStorageNode(blobber event.Blobber) (storageNodeResponse, error) {
	maxOfferDuration, err := time.ParseDuration(blobber.MaxOfferDuration)
	if err != nil {
		return storageNodeResponse{}, err
	}
	challengeCompletionTime, err := time.ParseDuration(blobber.ChallengeCompletionTime)
	if err != nil {
		return storageNodeResponse{}, err
	}
	return storageNodeResponse{
		StorageNode: StorageNode{
			Provider: provider.Provider{
				LastHealthCheck: common.Timestamp(blobber.LastHealthCheck),
			},
			ID:      blobber.BlobberID,
			BaseURL: blobber.BaseURL,
			Geolocation: StorageNodeGeolocation{
				Latitude:  blobber.Latitude,
				Longitude: blobber.Longitude,
			},
			Terms: Terms{
				ReadPrice:               state.Balance(blobber.ReadPrice),
				WritePrice:              state.Balance(blobber.WritePrice),
				MinLockDemand:           blobber.MinLockDemand,
				MaxOfferDuration:        maxOfferDuration,
				ChallengeCompletionTime: challengeCompletionTime,
			},
			Capacity: blobber.Capacity,
			Used:     blobber.Used,
			StakePoolSettings: stakepool.StakePoolSettings{
				DelegateWallet:  blobber.DelegateWallet,
				MinStake:        state.Balance(blobber.MinStake),
				MaxStake:        state.Balance(blobber.MaxStake),
				MaxNumDelegates: blobber.NumDelegates,
				ServiceCharge:   blobber.ServiceCharge,
			},
			Information: Info{
				Name:        blobber.Name,
				WebsiteUrl:  blobber.WebsiteUrl,
				LogoUrl:     blobber.LogoUrl,
				Description: blobber.Description,
			},
		},
		TotalStake: blobber.TotalStake,
	}, nil
}

func validatorTableToValidatorNode(val event.Validator) ValidationNode {
	return ValidationNode{
		Provider: provider.Provider{
			LastHealthCheck: common.Timestamp(val.LastHealthCheck),
			IsShutDown:      val.IsShutDown,
			IsKilled:        val.IsKilled,
		},
		ID:      val.ValidatorID,
		BaseURL: val.BaseUrl,
		StakePoolSettings: stakepool.StakePoolSettings{
			DelegateWallet:  val.DelegateWallet,
			MinStake:        val.MinStake,
			MaxStake:        val.MaxStake,
			MaxNumDelegates: val.NumDelegates,
			ServiceCharge:   val.ServiceCharge,
		},
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

// Deprecated

// GetBlobbersHandler returns list of all blobbers alive (e.g. excluding
// blobbers with zero capacity).
func (ssc *StorageSmartContract) GetBlobbersHandlerDeprecated(ctx context.Context,
	params url.Values, balances cstate.StateContextI) (interface{}, error) {

	blobbers, err := ssc.getBlobbersList(balances)
	if err != nil {
		return nil, smartcontract.NewErrNoResourceOrErrInternal(err, true, "can't get blobbers list")
	}
	return blobbers, nil
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

	sn, err := blobberTableToStorageNode(*blobber)
	if err != nil {
		return ssc.GetBlobberHandlerDepreciated(ctx, params, balances)
	}
	return sn, err
}

func (ssc *StorageSmartContract) GetStatus(
	ctx context.Context,
	params url.Values,
	balances cstate.StateContextI,
) (resp interface{}, err error) {
	var providerID = params.Get("id")
	if providerID == "" {
		return nil, common.NewErrBadRequest("missing 'blobber_id' URL query parameter")
	}
	var providerType = params.Get("type")
	if len(providerType) == 0 {
		providerType = spenum.Blobber.String()
	}

	if balances.GetEventDB() == nil {
		return ssc.GetBlobberHandlerDepreciated(ctx, params, balances)
	}

	var conf *Config
	conf, err = ssc.getConfig(balances, false)
	if err != nil && err != util.ErrValueNotPresent {
		return nil, smartcontract.NewErrNoResourceOrErrInternal(err, true, cantGetConfigErrMsg)
	}

	var status provider.StatusInfo
	switch providerType {
	case spenum.Blobber.String():
		blobber, err := balances.GetEventDB().GetBlobber(providerID)
		if err != nil {
			return ssc.GetBlobberHandlerDepreciated(ctx, params, balances)
		}
		sn, err := blobberTableToStorageNode(*blobber)
		if err != nil {
			return nil, err
		}
		status.Status, status.Reason = sn.Status(common.Timestamp(time.Now().Second()), conf)
	case spenum.Validator.String():
		validator, err := balances.GetEventDB().GetValidatorByValidatorID(providerID)
		if err != nil {
			return nil, err
		}
		val := validatorTableToValidatorNode(validator)
		status.Status, status.Reason = val.Status(common.Timestamp(time.Now().Second()), conf)
	default:
		return nil, common.NewErrBadRequest("invalid provider type %v", providerType)
	}

	return status, nil
}

// GetBlobberCountHandler returns Blobber count from its individual stored value.
func (ssc *StorageSmartContract) GetBlobberCountHandler(
	ctx context.Context,
	params url.Values,
	balances cstate.StateContextI,
) (resp interface{}, err error) {
	blobberCount, err := balances.GetEventDB().GetBlobberCount()
	if err != nil {
		return nil, fmt.Errorf("Error while geting the blobber count")
	}
	return map[string]int64{
		"count": blobberCount,
	}, nil
}

// GetBlobberTotalStakesHandler returns blobber total stake
func (ssc *StorageSmartContract) GetBlobberTotalStakesHandler(
	ctx context.Context,
	params url.Values,
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
	ctx context.Context,
	params url.Values,
	balances cstate.StateContextI,
) (resp interface{}, err error) {
	if balances.GetEventDB() == nil {
		return nil, fmt.Errorf("Unable to connect to eventdb database")
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
		return ssc.GetBlobbersHandlerDeprecated(ctx, params, balances)
	}
	blobbers, err := balances.GetEventDB().GetBlobbers()
	if err != nil || len(blobbers) == 0 {
		return ssc.GetBlobbersHandlerDeprecated(ctx, params, balances)
	}

	var sns storageNodesResponse
	for _, blobber := range blobbers {
		sn, err := blobberTableToStorageNode(blobber)
		if err != nil {
			return ssc.GetBlobbersHandlerDeprecated(ctx, params, balances)
		}
		sns.Nodes = append(sns.Nodes, sn)
	}
	return sns, nil
}

func (msc *StorageSmartContract) GetTransactionByHashHandler(
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
	return nil, errors.New("No filter selected")
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
	var creationDate = common.Timestamp(time.Now().Unix())

	allocData := params.Get("allocation_data")
	var request newAllocationRequest
	if err = request.decode([]byte(allocData)); err != nil {
		return "", common.NewErrInternal("can't decode allocation request", err.Error())
	}

	var allBlobbersList *StorageNodes
	allBlobbersList, err = ssc.getBlobbersList(balances)
	if err != nil {
		return "", common.NewErrInternal("can't get blobbers list", err.Error())
	}
	if len(allBlobbersList.Nodes) == 0 {
		return "", common.NewErrInternal("can't get blobbers list",
			"no blobbers found")
	}

	var sa = request.storageAllocation()

	blobberNodes, bSize, err := ssc.selectBlobbers(
		creationDate, *allBlobbersList, sa, int64(creationDate), balances)
	if err != nil {
		return "", common.NewErrInternal("selecting blobbers", err.Error())
	}

	var gbSize = sizeInGB(bSize)
	var minLockDemand state.Balance
	for _, b := range blobberNodes {
		minLockDemand += b.Terms.minLockDemand(gbSize,
			sa.restDurationInTimeUnits(creationDate))
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

type BlobberOpenChallengesResponse struct {
	BlobberID                string            `json:"blobber_id"`
	ChallengeIDs             []string          `json:"challenge_ids"`
	LatestCompletedChallenge *StorageChallenge `json:"lastest_completed_challenge"` // TODO: fix typo with Blobber and gosdk
}

func (ssc *StorageSmartContract) OpenChallengeHandler(ctx context.Context, params url.Values, balances cstate.StateContextI) (interface{}, error) {
	blobberID := params.Get("blobber")

	// return "404", if blobber not registered
	blobber := StorageNode{ID: blobberID}
	if err := balances.GetTrieNode(blobber.GetKey(ssc.ID), &blobber); err != nil {
		return "", smartcontract.NewErrNoResourceOrErrInternal(err, true, "can't find blobber")
	}

	rsp := &BlobberOpenChallengesResponse{
		BlobberID:    blobberID,
		ChallengeIDs: []string{},
	}

	// return "200" with empty list, if no challenges are found
	blobberChallenges := &BlobberChallenges{
		BlobberID: blobberID,
	}
	err := blobberChallenges.load(balances, ssc.ID)
	switch err {
	case util.ErrValueNotPresent:
		return rsp, nil
	case nil:
		lfb := balances.GetLatestFinalizedBlock()
		if lfb == nil {
			return nil, common.NewErrInternal("chain is not ready, could not get latest finalized block")
		}

		cct := getMaxChallengeCompletionTime()
		ocs := blobberChallenges.GetOpenChallengesNoExpire(lfb.CreationDate, cct)
		if len(ocs) > 0 {
			rsp.ChallengeIDs = make([]string, len(ocs))
			for i, oc := range ocs {
				rsp.ChallengeIDs[i] = oc.ID
			}
		}

		return rsp, nil
	default:
		return nil, common.NewErrInternal("fail to get blobber challenge", err.Error())
	}
}

func (ssc *StorageSmartContract) GetChallengeHandler(ctx context.Context, params url.Values, balances cstate.StateContextI) (retVal interface{}, retErr error) {
	defer func() {
		if retErr != nil {
			logging.Logger.Error("/getchallenge failed with error - " + retErr.Error())
		}
	}()
	blobberID := params.Get("blobber")
	blobberChallenges := &BlobberChallenges{}
	blobberChallenges.BlobberID = blobberID

	err := balances.GetTrieNode(blobberChallenges.GetKey(ssc.ID), blobberChallenges)
	if err != nil {
		return "", smartcontract.NewErrNoResourceOrErrInternal(err, true, "can't get blobber challenge")
	}

	challengeID := params.Get("challenge")
	if _, ok := blobberChallenges.ChallengesMap[challengeID]; !ok {
		return nil, common.NewErrBadRequest("can't find challenge with provided 'challenge' param")
	}

	challenge, err := ssc.getStorageChallenge(challengeID, balances)
	if err != nil {
		return "", smartcontract.NewErrNoResourceOrErrInternal(err, true, "can't get storage challenge")
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
	stat.UnstakeTotal = state.Balance(blobber.UnstakeTotal)
	stat.Capacity = blobber.Capacity
	stat.WritePrice = state.Balance(blobber.WritePrice)
	stat.OffersTotal = state.Balance(blobber.OffersTotal)
	stat.Delegate = make([]delegatePoolStat, 0, len(delegatePools))
	stat.Settings = stakepool.StakePoolSettings{
		DelegateWallet:  blobber.DelegateWallet,
		MinStake:        state.Balance(blobber.MinStake),
		MaxStake:        state.Balance(blobber.MaxStake),
		MaxNumDelegates: blobber.NumDelegates,
		ServiceCharge:   blobber.ServiceCharge,
	}
	stat.Rewards = state.Balance(blobber.Reward)
	for _, dp := range delegatePools {
		dpStats := delegatePoolStat{
			ID:           dp.PoolID,
			Balance:      state.Balance(dp.Balance),
			DelegateID:   dp.DelegateID,
			Rewards:      state.Balance(dp.Reward),
			Status:       spenum.PoolStatus(dp.Status).String(),
			TotalReward:  state.Balance(dp.TotalReward),
			TotalPenalty: state.Balance(dp.TotalPenalty),
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
	if ssc != nil {
		storageNodes, err := ssc.getBlobbersList(balances)
		if err != nil {
			return 0, fmt.Errorf("error from getBlobbersList in GetTotalData: %v", err)
		}

		var totalSavedData int64
		for _, sn := range storageNodes.Nodes {
			totalSavedData += sn.SavedData
		}

		return totalSavedData, nil
	}

	return 0, fmt.Errorf("storageSmartContract is nil")
}
