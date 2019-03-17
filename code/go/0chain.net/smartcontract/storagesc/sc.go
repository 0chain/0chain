package storagesc

import (
	"net/url"
	"encoding/json"
	"math/rand"
	"context"
	"time"
	"sort"

	"go.uber.org/zap"

	"0chain.net/chaincore/block"
	c_state "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/client"
	"0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/smartcontractstate"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"

	. "0chain.net/core/logging"
)

const (
	ADDRESS = "6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7"
)

type StorageSmartContract struct {
	*smartcontractinterface.SmartContract
}

func (ssc *StorageSmartContract) AllocationStatsHandler(ctx context.Context, params url.Values) (interface{}, error){
	allocationID := params.Get("allocation")
	allocationObj := &StorageAllocation{}
	allocationObj.ID = allocationID

	allocationBytes, err := ssc.DB.GetNode(allocationObj.GetKey())
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(allocationBytes, allocationObj)
	return allocationObj, err
}

func (ssc *StorageSmartContract) LatestReadMarkerHandler (ctx context.Context, params url.Values) (interface{}, error){
	clientID := params.Get("client")
	blobberID := params.Get("blobber")
	var commitRead ReadConnection
	commitRead.ReadMarker = &ReadMarker{BlobberID: blobberID, ClientID: clientID}

	commitReadBytes, err := ssc.DB.GetNode(commitRead.GetKey())
	if err != nil {
		return nil, err
	}
	if commitReadBytes == nil {
		return make(map[string]string), nil
	}
	err = commitRead.Decode(commitReadBytes)

	return commitRead.ReadMarker, err

}

func (ssc *StorageSmartContract) OpenChallengeHandler (ctx context.Context, params url.Values) (interface{}, error){
	blobberID := params.Get("blobber")
	var blobberChallengeObj BlobberChallenge
	blobberChallengeObj.BlobberID = blobberID

	blobberChallengeBytes, err := ssc.DB.GetNode(blobberChallengeObj.GetKey())
	if err != nil {
		return "", common.NewError("blobber_challenge_read_err", "Error reading blobber challenge from DB. " + err.Error())
	}
	err = blobberChallengeObj.Decode(blobberChallengeBytes)
	for k,v := range blobberChallengeObj.ChallengeMap {
		if v.Response != nil {
			delete(blobberChallengeObj.ChallengeMap, k)
		}
	}
	return &blobberChallengeObj, err
}

func (ssc *StorageSmartContract) SetSC(sc *smartcontractinterface.SmartContract, bcContext smartcontractinterface.BCContextI) {
	ssc.SmartContract = sc
	ssc.SmartContract.RestHandlers["/allocation"] = ssc.AllocationStatsHandler
	ssc.SmartContract.RestHandlers["/latestreadmarker"] = ssc.LatestReadMarkerHandler
	ssc.SmartContract.RestHandlers["/openchallenges"] = ssc.OpenChallengeHandler
}

type ChallengeResponse struct {
	ID                string              `json:"challenge_id"`
	ValidationTickets []*ValidationTicket `json:"validation_tickets"`
}

func (sc *StorageSmartContract) VerifyChallenge(t *transaction.Transaction, input []byte) (string, error) {
	var challengeResponse ChallengeResponse
	err := json.Unmarshal(input, &challengeResponse)
	if err != nil {
		return "", err
	}
	if len(challengeResponse.ID) == 0 || len(challengeResponse.ValidationTickets) == 0 {
		return "", common.NewError("invalid_parameters", "Invalid parameters to challenge response")
	}

	var blobberChallengeObj BlobberChallenge
	blobberChallengeObj.BlobberID = t.ClientID

	blobberChallengeBytes, err := sc.DB.GetNode(blobberChallengeObj.GetKey())
	if err != nil {
		return "", common.NewError("blobber_challenge_read_err", "Error reading blobber challenge from DB")
	}
	if blobberChallengeBytes == nil {
		return "", common.NewError("invalid_parameters", "Cannot find the blobber challenge entity with ID "+t.ClientID)
	} 

	err = blobberChallengeObj.Decode(blobberChallengeBytes)
	if err != nil {
		return "", common.NewError("blobber_challenge_decode_error", "Error decoding the blobber challenge")
	}

	challengeRequest:= blobberChallengeObj.ChallengeMap[challengeResponse.ID]
	
	if challengeRequest == nil {
		return "", common.NewError("invalid_parameters", "Cannot find the challenge with ID "+challengeResponse.ID)
	}
	
	if challengeRequest.Blobber.ID != t.ClientID {
		return "", common.NewError("invalid_parameters", "Challenge response should be submitted by the same blobber as the challenge request")
	}

	if challengeRequest.Response != nil {
		return "Challenge Already redeemed by Blobber", nil
	}

	numSuccess := 0
	numFailure := 0
	for _, vt := range challengeResponse.ValidationTickets {
		if vt != nil {
			ok, err := vt.VerifySign()
			if !ok || err != nil {
				continue
			}
			if vt.Result {
				numSuccess++
			} else {
				numFailure++
			}
		}
	}

	allocationObj := &StorageAllocation{}
	allocationObj.ID = challengeRequest.AllocationID

	allocationBytes, err := sc.DB.GetNode(allocationObj.GetKey())
	if allocationBytes == nil || err != nil {
		return "", common.NewError("invalid_allocation", "Client state has invalid allocations")
	}

	allocationObj.Decode(allocationBytes)

	

	if numSuccess > (len(challengeRequest.Validators) / 2) {
		challengeRequest.Response = &challengeResponse
		//delete(blobberChallengeObj.ChallengeMap, challengeResponse.ID)
		allocationObj.LastestClosedChallengeTxn = challengeRequest.ID
		allocationObj.ClosedChallenges++
		allocationObj.OpenChallenges--
		defer sc.DB.PutNode(allocationObj.GetKey(), allocationObj.Encode())
		sc.DB.PutNode(blobberChallengeObj.GetKey(), blobberChallengeObj.Encode())
		return "Challenge Passed by Blobber", nil
	}

	if numFailure > (len(challengeRequest.Validators) / 2) {
		//delete(blobberChallengeObj.ChallengeMap, challengeResponse.ID)
		challengeRequest.Response = &challengeResponse
		allocationObj.LastestClosedChallengeTxn = challengeRequest.ID
		allocationObj.ClosedChallenges++
		allocationObj.OpenChallenges--
		defer sc.DB.PutNode(allocationObj.GetKey(), allocationObj.Encode())
		sc.DB.PutNode(blobberChallengeObj.GetKey(), blobberChallengeObj.Encode())
		return "Challenge Failed by Blobber", nil
	}

	return "", common.NewError("not_enough_validations", "Not enough validations for the challenge")
}

func (sc *StorageSmartContract) AddChallenge(t *transaction.Transaction, b *block.Block, input []byte) (string, error) {
	
	validatorList, _ := sc.getValidatorsList()
	
	if len(validatorList) == 0 {
		return "", common.NewError("no_validators", "Not enough validators for the challenge")
	}

	foundValidator := false
	for _, validator := range validatorList {
		if validator.ID == t.ClientID {
			foundValidator = true
			break
		}
	}
	
	if !foundValidator {
		return "", common.NewError("invalid_challenge_request", "Challenge can be requested only by validators")
	}
	
	var storageChallenge StorageChallenge
	storageChallenge.ID = t.Hash
	
	allocationList, err := sc.getAllAllocationsList()
	if err != nil {
		return "", common.NewError("adding_challenge_error", "Error gettting the allocation list. "+err.Error())
	}
	if len(allocationList) == 0 {
		return "", common.NewError("adding_challenge_error", "No allocations at this time")
	}

	rand.Seed(b.RoundRandomSeed)
	allocationIndex := rand.Int63n(int64(len(allocationList)))
	allocationKey := allocationList[allocationIndex]

	allocationObj := &StorageAllocation{}
	allocationObj.ID = allocationKey

	allocationBytes, err := sc.DB.GetNode(allocationObj.GetKey())
	if allocationBytes == nil || err != nil {
		return "", common.NewError("invalid_allocation", "Client state has invalid allocations")
	}

	allocationObj.Decode(allocationBytes)
	sort.SliceStable(allocationObj.Blobbers, func(i, j int) bool {
		return allocationObj.Blobbers[i].ID < allocationObj.Blobbers[j].ID
	})
	storageChallenge.Validators = validatorList
	storageChallenge.Blobber = allocationObj.Blobbers[rand.Intn(len(allocationObj.Blobbers))]
	storageChallenge.RandomNumber = b.RoundRandomSeed
	storageChallenge.AllocationID = allocationObj.ID

	blobberAllocation := &BlobberAllocation{}
	blobberAllocation.BlobberID = storageChallenge.Blobber.ID
	blobberAllocation.AllocationID = allocationObj.ID

	blobberAllocationBytes, err := sc.DB.GetNode(blobberAllocation.GetKey())
	if blobberAllocationBytes == nil || err != nil {
		return "", common.NewError("invalid_parameters", "Blobber is not part of the allocation. Could not find blobber")
	}

	err = blobberAllocation.Decode(blobberAllocationBytes)
	if err != nil {
		return "", common.NewError("blobber_allocation_decode", "Blobber Allocation decode error "+err.Error())
	}
	if len(blobberAllocation.AllocationRoot) == 0 {
		return "", common.NewError("blobber_no_wm", "Blobber has no write marker committed.")
	}

	storageChallenge.AllocationRoot = blobberAllocation.AllocationRoot

	var blobberChallengeObj BlobberChallenge
	blobberChallengeObj.BlobberID = storageChallenge.Blobber.ID

	blobberChallengeBytes, err := sc.DB.GetNode(blobberChallengeObj.GetKey())
	if err != nil {
		return "", common.NewError("blobber_challenge_read_err", "Error reading blobber challenge from DB")
	}
	if blobberChallengeBytes == nil {
		blobberChallengeObj.ChallengeMap = make(map[string]*StorageChallenge)
	} else {
		err = blobberChallengeObj.Decode(blobberChallengeBytes)
		if err != nil {
			return "", common.NewError("blobber_challenge_decode_error", "Error decoding the blobber challenge")
		}
	}
	storageChallenge.Created = t.CreationDate
	blobberChallengeObj.ChallengeMap[storageChallenge.ID] = &storageChallenge

	sc.DB.PutNode(blobberChallengeObj.GetKey(), blobberChallengeObj.Encode())
	allocationObj.OpenChallenges++
	sc.DB.PutNode(allocationObj.GetKey(), allocationObj.Encode())
	challengeBytes, err := json.Marshal(storageChallenge)
	return string(challengeBytes), err
}

func (sc *StorageSmartContract) CommitBlobberRead(t *transaction.Transaction, input []byte) (string, error) {
	var commitRead ReadConnection
	err := commitRead.Decode(input)
	if err != nil {
		return "", err
	}

	lastBlobberClientReadBytes, err := sc.DB.GetNode(commitRead.GetKey())
	if err != nil {
		return "", common.NewError("rm_read_error", "Error reading the read marker for the blobber and client")
	}
	lastCommittedRM := &ReadConnection{}
	if lastBlobberClientReadBytes != nil {
		lastCommittedRM.Decode(lastBlobberClientReadBytes)
	}

	err = commitRead.ReadMarker.Verify(lastCommittedRM.ReadMarker)
	if err != nil {
		return "", common.NewError("invalid_read_marker", "Invalid read marker." + err.Error())
	}
	sc.DB.PutNode(commitRead.GetKey(), input)
	return "success", nil
}

func (sc *StorageSmartContract) CommitBlobberConnection(t *transaction.Transaction, input []byte) (string, error) {
	var commitConnection BlobberCloseConnection
	err := json.Unmarshal(input, &commitConnection)
	if err != nil {
		return "", err
	}

	if !commitConnection.Verify() {
		return "", common.NewError("invalid_parameters", "Invalid input")
	}

	if commitConnection.WriteMarker.BlobberID != t.ClientID {
		return "", common.NewError("invalid_parameters", "Invalid Blobber ID for closing connection. Write marker not for this blobber")
	}

	allocationObj := &StorageAllocation{}
	allocationObj.ID = commitConnection.WriteMarker.AllocationID
	allocationBytes, err := sc.DB.GetNode(allocationObj.GetKey())

	// allocationObj, ok := allocationRequestMap[openConnection.AllocationID]
	if allocationBytes == nil || err != nil {
		return "", common.NewError("invalid_parameters", "Invalid allocation ID")
	}

	err = allocationObj.Decode(allocationBytes)
	if allocationBytes == nil || err != nil {
		return "", common.NewError("invalid_parameters", "Invalid allocation ID. Failed to decode from DB")
	}

	if allocationObj.Owner != commitConnection.WriteMarker.ClientID {
		return "", common.NewError("invalid_parameters", "Write marker has to be by the same client as owner of the allocation")
	}

	blobberAllocation := &BlobberAllocation{}
	blobberAllocation.BlobberID = t.ClientID
	blobberAllocation.AllocationID = commitConnection.WriteMarker.AllocationID

	blobberAllocationBytes, err := sc.DB.GetNode(blobberAllocation.GetKey())
	if blobberAllocationBytes == nil || err != nil {
		return "", common.NewError("invalid_parameters", "Blobber is not part of the allocation. Could not find blobber")
	}

	err = blobberAllocation.Decode(blobberAllocationBytes)
	if err != nil {
		return "", common.NewError("blobber_allocation_decode", "Blobber Allocation decode error "+err.Error())
	}

	if blobberAllocation.AllocationRoot != commitConnection.PrevAllocationRoot {
		return "", common.NewError("invalid_parameters", "Previous allocation root does not match the latest allocation root")
	}

	if blobberAllocation.UsedSize+commitConnection.WriteMarker.Size > blobberAllocation.Size {
		return "", common.NewError("invalid_parameters", "Size for blobber allocation exceeded maximum")
	}

	if !commitConnection.WriteMarker.VerifySignature(allocationObj.OwnerPublicKey) {
		return "", common.NewError("invalid_parameters", "Invalid signature for write marker")
	}

	blobberAllocation.AllocationRoot = commitConnection.AllocationRoot
	blobberAllocation.LastWriteMarker = commitConnection.WriteMarker
	blobberAllocation.UsedSize += commitConnection.WriteMarker.Size

	buffBlobberAllocation := blobberAllocation.Encode()
	sc.DB.PutNode(blobberAllocation.GetKey(), buffBlobberAllocation)

	allocationObj.UsedSize += commitConnection.WriteMarker.Size
	allocationObj.NumWrites++
	sc.DB.PutNode(allocationObj.GetKey(), allocationObj.Encode())

	return string(buffBlobberAllocation), nil
}

func (sc *StorageSmartContract) getAllocationsList(clientID string) ([]string, error) {
	var allocationList = make([]string, 0)
	var clientAlloc ClientAllocation
	clientAlloc.ClientID = clientID
	allocationListBytes, err := sc.DB.GetNode(clientAlloc.GetKey())
	if err != nil {
		return nil, common.NewError("getAllocationsList_failed", "Failed to retrieve existing allocations list")
	}
	if allocationListBytes == nil {
		return allocationList, nil
	}
	err = json.Unmarshal(allocationListBytes, &clientAlloc)
	if err != nil {
		return nil, common.NewError("getAllocationsList_failed", "Failed to retrieve existing allocations list")
	}
	return clientAlloc.Allocations, nil
}

func (sc *StorageSmartContract) getAllAllocationsList() ([]string, error) {
	var allocationList = make([]string, 0)
	
	allocationListBytes, err := sc.DB.GetNode(ALL_ALLOCATIONS_KEY)
	if err != nil {
		return nil, common.NewError("getAllAllocationsList_failed", "Failed to retrieve existing allocations list")
	}
	if allocationListBytes == nil {
		return allocationList, nil
	}
	err = json.Unmarshal(allocationListBytes, &allocationList)
	if err != nil {
		return nil, common.NewError("getAllAllocationsList_failed", "Failed to retrieve existing allocations list")
	}
	sort.SliceStable(allocationList, func(i, j int) bool {
		return allocationList[i] < allocationList[j]
	})
	return allocationList, nil
}


func (sc *StorageSmartContract) getBlobbersList() ([]StorageNode, error) {
	var allBlobbersList = make([]StorageNode, 0)
	allBlobbersBytes, err := sc.DB.GetNode(ALL_BLOBBERS_KEY)
	if err != nil {
		return nil, common.NewError("getBlobbersList_failed", "Failed to retrieve existing blobbers list")
	}
	if allBlobbersBytes == nil {
		return allBlobbersList, nil
	}
	err = json.Unmarshal(allBlobbersBytes, &allBlobbersList)
	if err != nil {
		return nil, common.NewError("getBlobbersList_failed", "Failed to retrieve existing blobbers list")
	}
	sort.SliceStable(allBlobbersList, func(i, j int) bool {
		return allBlobbersList[i].ID < allBlobbersList[j].ID
	})
	return allBlobbersList, nil
}

func (sc *StorageSmartContract) getValidatorsList() ([]ValidationNode, error) {
	var allValidatorsList = make([]ValidationNode, 0)
	allValidatorsBytes, err := sc.DB.GetNode(ALL_VALIDATORS_KEY)
	if err != nil {
		return nil, common.NewError("getValidatorsList_failed", "Failed to retrieve existing validators list")
	}
	if allValidatorsBytes == nil {
		return allValidatorsList, nil
	}
	err = json.Unmarshal(allValidatorsBytes, &allValidatorsList)
	if err != nil {
		return nil, common.NewError("getValidatorsList_failed", "Failed to retrieve existing validators list")
	}
	return allValidatorsList, nil
}

func (sc *StorageSmartContract) addAllocation(allocation *StorageAllocation) (string, error) {
	allocationList, err := sc.getAllocationsList(allocation.Owner)
	if err != nil {
		return "", common.NewError("add_allocation_failed", "Failed to get allocation list"+err.Error())
	}
	allAllocationList, err := sc.getAllAllocationsList()
	if err != nil {
		return "", common.NewError("add_allocation_failed", "Failed to get allocation list"+err.Error())
	}

	allocationBytes, _ := sc.DB.GetNode(allocation.GetKey())
	if allocationBytes == nil {
		allocationList = append(allocationList, allocation.ID)
		allAllocationList = append(allAllocationList, allocation.ID)
		var clientAllocation ClientAllocation
		clientAllocation.ClientID = allocation.Owner
		clientAllocation.Allocations = allocationList
		
		allAllocationBytes, _ := json.Marshal(allAllocationList)
		sc.DB.PutNode(ALL_ALLOCATIONS_KEY, allAllocationBytes)
		sc.DB.PutNode(clientAllocation.GetKey(), clientAllocation.Encode())
		sc.DB.PutNode(allocation.GetKey(), allocation.Encode())
		Logger.Info("Adding allocation")
	}

	buff := allocation.Encode()
	return string(buff), nil
}

func (sc *StorageSmartContract) AddBlobber(t *transaction.Transaction, input []byte) (string, error) {
	allBlobbersList, err := sc.getBlobbersList()
	if err != nil {
		return "", common.NewError("add_blobber_failed", "Failed to get blobber list"+err.Error())
	}
	var newBlobber StorageNode
	err = newBlobber.Decode(input) //json.Unmarshal(input, &newBlobber)
	if err != nil {
		return "", err
	}
	newBlobber.ID = t.ClientID
	newBlobber.PublicKey = t.PublicKey
	blobberBytes, _ := sc.DB.GetNode(newBlobber.GetKey())
	if blobberBytes == nil {
		allBlobbersList = append(allBlobbersList, newBlobber)
		allBlobbersBytes, _ := json.Marshal(allBlobbersList)
		sc.DB.PutNode(ALL_BLOBBERS_KEY, allBlobbersBytes)
		sc.DB.PutNode(newBlobber.GetKey(), newBlobber.Encode())
		Logger.Info("Adding blobber to known list of blobbers")
	}

	buff := newBlobber.Encode()
	return string(buff), nil
}

func (sc *StorageSmartContract) AddValidator(t *transaction.Transaction, input []byte) (string, error) {
	allValidatorsList, err := sc.getValidatorsList()
	if err != nil {
		return "", common.NewError("add_validator_failed", "Failed to get validator list."+err.Error())
	}
	var newValidator ValidationNode
	err = newValidator.Decode(input) //json.Unmarshal(input, &newBlobber)
	if err != nil {
		return "", err
	}
	newValidator.ID = t.ClientID
	newValidator.PublicKey = t.PublicKey
	blobberBytes, _ := sc.DB.GetNode(newValidator.GetKey())
	if blobberBytes == nil {
		allValidatorsList = append(allValidatorsList, newValidator)
		allValidatorsBytes, _ := json.Marshal(allValidatorsList)
		sc.DB.PutNode(ALL_VALIDATORS_KEY, allValidatorsBytes)
		sc.DB.PutNode(newValidator.GetKey(), newValidator.Encode())
		Logger.Info("Adding validator to known list of validators")
	}

	buff := newValidator.Encode()
	return string(buff), nil
}

func shuffleStorageNodes(vals []StorageNode) {
	r := rand.New(rand.NewSource(time.Now().Unix()))
	// We start at the end of the slice, inserting our random
	// values one at a time.
	for n := len(vals); n > 0; n-- {
	  randIndex := r.Intn(n)
	  // We swap the value at index n-1 and the random index
	  // to move our randomly chosen value to the end of the
	  // slice, and to move the value that was at n-1 into our
	  // unshuffled portion of the slice.
	  vals[n-1], vals[randIndex] = vals[randIndex], vals[n-1]
	}
  }

func (sc *StorageSmartContract) NewAllocationRequest(t *transaction.Transaction, input []byte) (string, error) {
	allBlobbersList, err := sc.getBlobbersList()
	if err != nil {
		return "", common.NewError("allocation_creation_failed", "No Blobbers registered. Failed to create a storage allocation")
	}

	if len(allBlobbersList) == 0 {
		return "", common.NewError("allocation_creation_failed", "No Blobbers registered. Failed to create a storage allocation")
	}

	if len(t.ClientID) == 0 {
		return "", common.NewError("allocation_creation_failed", "Invalid client in the transaction. No public key found")
	}

	clientPublicKey := t.PublicKey
	if len(t.PublicKey) == 0 {
		ownerClient, err := client.GetClient(common.GetRootContext(), t.ClientID)
		if err != nil || ownerClient == nil || len(ownerClient.PublicKey) == 0 {
			return "", common.NewError("invalid_client", "Invalid Client. Not found with miner")
		}
		clientPublicKey = ownerClient.PublicKey
	}

	var allocationRequest StorageAllocation

	err = allocationRequest.Decode(input)
	if err != nil {
		return "", common.NewError("allocation_creation_failed", "Failed to create a storage allocation")
	}

	if allocationRequest.Size > 0 && allocationRequest.DataShards > 0 {
		size := allocationRequest.DataShards + allocationRequest.ParityShards

		if len(allBlobbersList) < size {
			return "", common.NewError("not_enough_blobbers", "Not enough blobbers to honor the allocation")
		}
		//shuffleStorageNodes(allBlobbersList)
		allocatedBlobbers := make([]*StorageNode, 0)

		blobberAllocationKeys := make([]smartcontractstate.Key, 0)
		blobberAllocationValues := make([]smartcontractstate.Node, 0)
		for i := 0; i < size; i++ {
			blobberNode := allBlobbersList[i]

			var blobberAllocation BlobberAllocation
			blobberAllocation.Size = (allocationRequest.Size + int64(size-1)) / int64(size)
			blobberAllocation.UsedSize = 0
			blobberAllocation.AllocationRoot = ""
			blobberAllocation.AllocationID = t.Hash
			blobberAllocation.BlobberID = blobberNode.ID

			//blobberAllocations = append(blobberAllocations, blobberAllocation)
			blobberAllocationKeys = append(blobberAllocationKeys, blobberAllocation.GetKey())
			buff := blobberAllocation.Encode()
			blobberAllocationValues = append(blobberAllocationValues, buff)
			allocatedBlobbers = append(allocatedBlobbers, &blobberNode)
		}
		sort.SliceStable(allocatedBlobbers, func(i, j int) bool {
			return allocatedBlobbers[i].ID < allocatedBlobbers[j].ID
		})
		allocationRequest.Blobbers = allocatedBlobbers
		allocationRequest.ID = t.Hash
		allocationRequest.Owner = t.ClientID
		allocationRequest.OwnerPublicKey = clientPublicKey

		//allocationRequestMap[t.Hash] = allocationRequest
		Logger.Info("Length of the keys and values", zap.Any("keys", len(blobberAllocationKeys)), zap.Any("values", len(blobberAllocationValues)))
		err = sc.DB.MultiPutNode(blobberAllocationKeys, blobberAllocationValues)
		if err != nil {
			return "", common.NewError("allocation_request_failed", "Failed to store the blobber allocation stats")
		}
		buff, err := sc.addAllocation(&allocationRequest)
		if err != nil {
			return "", common.NewError("allocation_request_failed", "Failed to store the allocation request")
		}
		return buff, nil
	}
	return "", common.NewError("invalid_allocation_request", "Failed storage allocate")
}

func (sc *StorageSmartContract) Execute(t *transaction.Transaction, funcName string, input []byte, balances c_state.StateContextI) (string, error) {

	// if funcName == "challenge_response" {
	// 	resp, err := sc.VerifyChallenge(t, input)
	// 	if err != nil {
	// 		return "", err
	// 	}
	// 	return resp, nil
	// }

	// if funcName == "open_connection" {
	// 	resp, err := sc.OpenConnectionWithBlobber(t, input)
	// 	if err != nil {
	// 		return "", err
	// 	}
	// 	return resp, nil
	// }

	if funcName == "read_redeem" {
		resp, err := sc.CommitBlobberRead(t, input)
		if err != nil {
			return "", err
		}
		return resp, nil
	}

	if funcName == "commit_connection" {
		resp, err := sc.CommitBlobberConnection(t, input)
		if err != nil {
			return "", err
		}
		return resp, nil
	}

	if funcName == "new_allocation_request" {
		resp, err := sc.NewAllocationRequest(t, input)
		if err != nil {
			return "", err
		}
		return resp, nil
	}

	if funcName == "add_blobber" {
		resp, err := sc.AddBlobber(t, input)
		if err != nil {
			return "", err
		}
		return resp, nil
	}

	if funcName == "add_validator" {
		resp, err := sc.AddValidator(t, input)
		if err != nil {
			return "", err
		}
		return resp, nil
	}

	if funcName == "challenge_request" {
		resp, err := sc.AddChallenge(t, balances.GetBlock(), input)
		if err != nil {
			return "", err
		}
		return resp, nil
	}

	if funcName == "challenge_response" {
		resp, err := sc.VerifyChallenge(t, input)
		if err != nil {
			return "", err
		}
		return resp, nil
	}

	return "", common.NewError("invalid_storage_function_name", "Invalid storage function called")
}
