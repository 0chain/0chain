package storagesc

import (
	"encoding/json"

	"go.uber.org/zap"

	c_state "0chain.net/chain/state"
	"0chain.net/client"
	"0chain.net/common"
	"0chain.net/smartcontractinterface"
	"0chain.net/smartcontractstate"
	"0chain.net/transaction"
	"0chain.net/util"

	. "0chain.net/logging"
)

type StorageSmartContract struct {
	smartcontractinterface.SmartContract
}

type ChallengeResponse struct {
	Data        []byte       `json:"data_bytes"`
	WriteMarker *WriteMarker `json:"write_marker"`
	MerkleRoot  string       `json:"merkle_root"`
	MerklePath  *util.MTPath `json:"merkle_path"`
	CloseTxnID  string       `json:"close_txn_id"`
}

// func (sc *StorageSmartContract) VerifyChallenge(t *transaction.Transaction, input []byte) (string, error) {
// 	var challengeResponse ChallengeResponse
// 	err := json.Unmarshal(input, &challengeResponse)
// 	if err != nil {
// 		return "", err
// 	}
// 	if len(challengeResponse.CloseTxnID) == 0 || len(challengeResponse.Data) == 0 || challengeResponse.MerklePath == nil || len(challengeResponse.MerkleRoot) == 0 || challengeResponse.WriteMarker == nil {
// 		return "", common.NewError("invalid_parameters", "Invalid parameters to challenge response")
// 	}

// 	commitConnectionDBBytes, err := sc.DB.GetNode(smartcontractstate.Key("close_connection:" + challengeResponse.WriteMarker.DataID))
// 	if commitConnectionDBBytes == nil || err != nil {
// 		return "", common.NewError("invalid_parameters", "Cannot find the close connection for the Data ID "+challengeResponse.WriteMarker.DataID)
// 	}

// 	var closeConnection BlobberCloseConnection
// 	err = json.Unmarshal(commitConnectionDBBytes, &closeConnection)
// 	if err != nil {
// 		return "", common.NewError("close_connection_decode_error", "Invalid connection stored in the state. "+challengeResponse.WriteMarker.DataID)
// 	}

// 	if closeConnection.DataID != challengeResponse.WriteMarker.DataID {
// 		return "", common.NewError("invalid_parameters", "Invalid Write marker / Data ID sent for the challenge")
// 	}

// 	if closeConnection.MerkleRoot != challengeResponse.MerkleRoot {
// 		return "", common.NewError("invalid_parameters", "Invalid Merkle root sent for the challenge")
// 	}

// 	if closeConnection.WriteMarker.Signature != challengeResponse.WriteMarker.Signature {
// 		return "", common.NewError("invalid_parameters", "Invalid Write marker sent for the challenge")
// 	}
// 	if t.ClientID != challengeResponse.WriteMarker.BlobberID {
// 		return "", common.NewError("invalid_parameters", "Challenge response should be submitted by the same blobber as the write marker")
// 	}

// 	// var dataBytes64Encode bytes.Buffer
// 	// dataBytes64EncodeWriter := bufio.NewWriter(&dataBytes64Encode)
// 	// inputZlibBytes := bytes.NewBuffer(challengeResponse.Data)
// 	// zlibReader, err := zlib.NewReader(inputZlibBytes)
// 	// io.Copy(dataBytes64EncodeWriter, zlibReader)
// 	// zlibReader.Close()

// 	// var dataBytes bytes.Buffer
// 	// dataBytesWriter := bufio.NewWriter(&dataBytes)
// 	// base64Decoder := base64.NewDecoder(base64.StdEncoding, bytes.NewReader(dataBytes64Encode.Bytes()))
// 	// io.Copy(dataBytesWriter, base64Decoder)
// 	contentHash := encryption.Hash(challengeResponse.Data)
// 	merkleVerify := util.VerifyMerklePath(contentHash, challengeResponse.MerklePath, challengeResponse.MerkleRoot)
// 	if !merkleVerify {
// 		return "", common.NewError("challenge_failed", "Challenge failed since we could not verify the merkle tree")
// 	}
// 	return "Challenge Passed by Blobber", nil
// }

func (sc *StorageSmartContract) CommitBlobberRead(t *transaction.Transaction, input []byte) (string, error) {
	var commitRead ReadConnection
	err := commitRead.Decode(input)
	if err != nil {
		return "", err
	}

	allocationObj := &StorageAllocation{}
	allocationObj.ID = commitRead.ReadMarker.AllocationID
	allocationBytes, err := sc.DB.GetNode(allocationObj.GetKey())

	// allocationObj, ok := allocationRequestMap[openConnection.AllocationID]
	if allocationBytes == nil || err != nil {
		return "", common.NewError("invalid_parameters", "Invalid allocation ID")
	}

	err = allocationObj.Decode(allocationBytes)
	if allocationBytes == nil || err != nil {
		return "", common.NewError("invalid_parameters", "Invalid allocation ID. Failed to decode from DB")
	}

	blobberAllocation := &BlobberAllocation{}
	blobberAllocation.BlobberID = t.ClientID
	blobberAllocation.AllocationID = commitRead.ReadMarker.AllocationID

	blobberAllocationBytes, err := sc.DB.GetNode(blobberAllocation.GetKey())
	if blobberAllocationBytes == nil || err != nil {
		return "", common.NewError("invalid_parameters", "Blobber is not part of the allocation. Could not find blobber")
	}

	err = blobberAllocation.Decode(blobberAllocationBytes)
	if err != nil {
		return "", common.NewError("blobber_allocation_decode", "Blobber Allocation decode error "+err.Error())
	}

	lastBlobberClientReadBytes, err := sc.DB.GetNode(commitRead.GetKey())
	if err != nil {
		return "", common.NewError("rm_read_error", "Error reading the read marker for the blobber and client")
	}
	lastCommittedRM := &ReadConnection{}
	if lastBlobberClientReadBytes != nil {
		lastCommittedRM.Decode(lastBlobberClientReadBytes)
	}

	ok := commitRead.ReadMarker.Verify(lastCommittedRM.ReadMarker)
	if !ok {
		return "", common.NewError("invalid_read_marker", "Invalid read marker.")
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
	sc.DB.PutNode(allocationObj.GetKey(), allocationObj.Encode())

	return string(buffBlobberAllocation), nil
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
	return allBlobbersList, nil
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

		if len(allBlobbersList) < allocationRequest.DataShards+allocationRequest.ParityShards {
			size = len(allBlobbersList)
		}
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
		allocationRequest.Blobbers = allocatedBlobbers
		allocationRequest.ID = t.Hash
		allocationRequest.Owner = t.ClientID
		allocationRequest.OwnerPublicKey = clientPublicKey
		buff := allocationRequest.Encode()
		//allocationRequestMap[t.Hash] = allocationRequest
		Logger.Info("Length of the keys and values", zap.Any("keys", len(blobberAllocationKeys)), zap.Any("values", len(blobberAllocationValues)))
		err = sc.DB.MultiPutNode(blobberAllocationKeys, blobberAllocationValues)
		if err != nil {
			return "", common.NewError("allocation_request_failed", "Failed to store the blobber allocation stats")
		}
		err = sc.DB.PutNode(allocationRequest.GetKey(), buff)
		if err != nil {
			return "", common.NewError("allocation_request_failed", "Failed to store the allocation request")
		}
		return string(buff), nil
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

	return "", common.NewError("invalid_storage_function_name", "Invalid storage function called")
}
