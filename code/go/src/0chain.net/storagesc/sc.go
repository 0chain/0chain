package storagesc

import (
	"encoding/json"

	"go.uber.org/zap"

	"0chain.net/common"
	"0chain.net/encryption"
	"0chain.net/smartcontractinterface"
	"0chain.net/smartcontractstate"
	"0chain.net/transaction"
	"0chain.net/util"

	. "0chain.net/logging"
)

const Seperator = smartcontractinterface.Seperator

const (
	Unknown = iota
	Active
	Closed
)

type StorageConnection struct {
	ClientPublicKey string                      `json:"client_public_key"`
	AllocationID    string                      `json:"allocation_id"`
	Status          int                         `json:"status"`
	BlobberData     []*StorageConnectionBlobber `json:"blobber_data"`
}

type StorageConnectionBlobber struct {
	BlobberID         string `json:"blobber_id"`
	DataID            string `json:"data_id"`
	Size              int64  `json:"size"`
	MerkleRoot        string `json:"merkle_root"`
	OpenConnectionTxn string `json:"open_connection_txn"`
	AllocationID      string `json:"allocation_id"`
}

func (ba *StorageConnectionBlobber) Encode() []byte {
	buff, _ := json.Marshal(ba)
	return buff
}

func (ba *StorageConnectionBlobber) Decode(input []byte) error {
	err := json.Unmarshal(input, ba)
	if err != nil {
		return err
	}
	return nil
}

type StorageSmartContract struct {
	smartcontractinterface.SmartContract
}

type BlobberCloseConnection struct {
	DataID      string      `json:"data_id"`
	MerkleRoot  string      `json:"merkle_root"`
	Size        int64       `json:"size"`
	WriteMarker WriteMarker `json:"write_marker"`
}

type WriteMarker struct {
	DataID              string           `json:"data_id"`
	MerkleRoot          string           `json:"merkle_root"`
	IntentTransactionID string           `json:"intent_tx_id"`
	BlobberID           string           `json:"blobber_id"`
	Timestamp           common.Timestamp `json:"timestamp"`
	ClientID            string           `json:"client_id"`
	Signature           string           `json:"signature"`
}

type ChallengeResponse struct {
	Data        []byte       `json:"data_bytes"`
	WriteMarker *WriteMarker `json:"write_marker"`
	MerkleRoot  string       `json:"merkle_root"`
	MerklePath  *util.MTPath `json:"merkle_path"`
	CloseTxnID  string       `json:"close_txn_id"`
}

func generateClientBlobberKey(allocationID string, clientID string, blobberID string) string {
	return encryption.Hash(allocationID + Seperator + clientID + Seperator + blobberID)
}

func generateAllocationBlobberKey(allocationID string, blobberID string) string {
	return encryption.Hash(allocationID + Seperator + blobberID)
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

// func (sc *StorageSmartContract) OpenConnectionWithBlobber(t *transaction.Transaction, input []byte) (string, error) {
// 	var openConnection StorageConnection
// 	err := json.Unmarshal(input, &openConnection)
// 	if err != nil {
// 		return "", err
// 	}
// 	if len(openConnection.AllocationID) == 0 {
// 		return "", common.NewError("invalid_parameters", "Invalid ClientID, BlobberID or Allocation ID for opening connection.")
// 	}

// 	allocationBytes, err := sc.DB.GetNode(smartcontractstate.Key("allocation:" + openConnection.AllocationID))

// 	// allocationObj, ok := allocationRequestMap[openConnection.AllocationID]
// 	if allocationBytes == nil || err != nil {
// 		return "", common.NewError("invalid_parameters", "Invalid allocation ID")
// 	}

// 	allocationObj := &StorageAllocation{}
// 	err = allocationObj.Decode(allocationBytes)
// 	if allocationBytes == nil || err != nil {
// 		return "", common.NewError("invalid_parameters", "Invalid allocation ID. Failed to decode from DB")
// 	}

// 	if allocationObj.Owner != t.ClientID {
// 		return "", common.NewError("invalid_parameters", "Connection has to be opened by the same client as owner of the allocation")
// 	}

// 	if allocationObj.Expiration < common.Now() {
// 		return "", common.NewError("invalid_parameters", "Allocation is expired")
// 	}

// 	for _, blobberConnection := range openConnection.BlobberData {
// 		blobberAllocationKey := generateAllocationBlobberKey(openConnection.AllocationID, blobberConnection.BlobberID)
// 		if _, ok := allocationObj.Blobbers[blobberAllocationKey]; !ok {
// 			return "", common.NewError("invalid_parameters", "Blobber is not part of the allocation")
// 		}
// 		blobberAllocationBytes, err := sc.DB.GetNode(smartcontractstate.Key("blobber_allocation:" + blobberAllocationKey))
// 		if blobberAllocationBytes == nil || err != nil {
// 			return "", common.NewError("invalid_parameters", "Blobber is not part of the allocation. Could not find blobber in the DB")
// 		}
// 		blobberAllocation := &BlobberAllocation{}
// 		err = blobberAllocation.Decode(blobberAllocationBytes)
// 		if err != nil {
// 			return "", common.NewError("blobber_allocation_decode", "Blobber Allocation decode error"+err.Error())
// 		}
// 		if blobberAllocation.Capacity < blobberConnection.Size {
// 			return "", common.NewError("insufficient_capacity", "blobber does not have enough storage capacity to handle this request")
// 		}
// 		blobberConnection.OpenConnectionTxn = t.Hash
// 		blobberConnection.AllocationID = openConnection.AllocationID
// 		sc.DB.PutNode(smartcontractstate.Key("open_connection:"+blobberConnection.DataID), blobberConnection.Encode())
// 	}
// 	openConnection.ClientPublicKey = t.PublicKey

// 	buff, _ := json.Marshal(openConnection)
// 	return string(buff), nil
// }

// func (sc *StorageSmartContract) CloseConnectionWithBlobber(t *transaction.Transaction, input []byte) (string, error) {
// 	var commitConnection BlobberCloseConnection
// 	err := json.Unmarshal(input, &commitConnection)
// 	if err != nil {
// 		return "", err
// 	}

// 	if commitConnection.DataID != commitConnection.WriteMarker.DataID {
// 		return "", common.NewError("invalid_parameters", "Invalid Data ID for closing connection.")
// 	}

// 	if commitConnection.WriteMarker.BlobberID != t.ClientID {
// 		return "", common.NewError("invalid_parameters", "Invalid Blobber ID for closing connection. Write marker not for this blobber")
// 	}

// 	commitConnectionDBBytes, err := sc.DB.GetNode(smartcontractstate.Key("close_connection:" + commitConnection.DataID))
// 	if commitConnectionDBBytes != nil {
// 		return "", common.NewError("invalid_parameters", "Connection for the data id is already closed")
// 	}

// 	blobberConnectionBytes, err := sc.DB.GetNode(smartcontractstate.Key("open_connection:" + commitConnection.DataID))

// 	if blobberConnectionBytes == nil || err != nil {
// 		return "", common.NewError("invalid_parameters", "Not valid open connection for the data ID")
// 	}
// 	var blobberConnection StorageConnectionBlobber
// 	err = blobberConnection.Decode(blobberConnectionBytes)
// 	if err != nil {
// 		return "", common.NewError("invalid_parameters", "Unable to get the blobber connection object from DB")
// 	}

// 	if blobberConnection.BlobberID != commitConnection.WriteMarker.BlobberID {
// 		return "", common.NewError("invalid_parameters", "Connection was open for a different blobber")
// 	}

// 	if blobberConnection.OpenConnectionTxn != commitConnection.WriteMarker.IntentTransactionID {
// 		return "", common.NewError("invalid_parameters", "Write marker is not for the same open connection")
// 	}

// 	if blobberConnection.Size != commitConnection.Size {
// 		return "", common.NewError("invalid_parameters", "Size uploaded does not match the size on open connection")
// 	}

// 	allocationBytes, err := sc.DB.GetNode(smartcontractstate.Key("allocation:" + blobberConnection.AllocationID))

// 	// allocationObj, ok := allocationRequestMap[openConnection.AllocationID]
// 	if allocationBytes == nil || err != nil {
// 		return "", common.NewError("invalid_parameters", "Invalid allocation ID")
// 	}

// 	allocationObj := &StorageAllocation{}
// 	err = allocationObj.Decode(allocationBytes)
// 	if allocationBytes == nil || err != nil {
// 		return "", common.NewError("invalid_parameters", "Invalid allocation ID. Failed to decode from DB")
// 	}

// 	if allocationObj.Owner != commitConnection.WriteMarker.ClientID {
// 		return "", common.NewError("invalid_parameters", "Write marker has to be by the same client as owner of the allocation")
// 	}

// 	blobberAllocationKey := generateAllocationBlobberKey(blobberConnection.AllocationID, blobberConnection.BlobberID)
// 	blobberAllocationBytes, err := sc.DB.GetNode(smartcontractstate.Key("blobber_allocation:" + blobberAllocationKey))
// 	if blobberAllocationBytes == nil || err != nil {
// 		return "", common.NewError("invalid_parameters", "Blobber is not part of the allocation. Could not find blobber in the DB")
// 	}
// 	blobberAllocation := &BlobberAllocation{}
// 	err = blobberAllocation.Decode(blobberAllocationBytes)
// 	if err != nil {
// 		return "", common.NewError("blobber_allocation_decode", "Blobber Allocation decode error "+err.Error())
// 	}

// 	blobberAllocation.LatestCloseTxn = t.Hash
// 	blobberAllocation.Capacity -= blobberConnection.Size
// 	buffBlobberAllocation, _ := json.Marshal(blobberAllocation)
// 	sc.DB.PutNode(smartcontractstate.Key("blobber_allocation:"+blobberAllocationKey), buffBlobberAllocation)
// 	buff, _ := json.Marshal(commitConnection)
// 	sc.DB.PutNode(smartcontractstate.Key("close_connection:"+commitConnection.DataID), buff)
// 	sc.addToAllClosedConnectionsList(commitConnection)
// 	return string(buffBlobberAllocation), nil
// }

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

// func (sc *StorageSmartContract) addToAllClosedConnectionsList(closedConnection BlobberCloseConnection) error {
// 	var allClosedConnectionsList = make([]BlobberCloseConnection, 0)
// 	//sc.DB.PutNode()
// 	closedConnectionBytes, err := sc.DB.GetNode(smartcontractstate.Key("all_closed_connections"))
// 	if err != nil {
// 		return common.NewError("addToAllClosedConnectionsList_failed", "Failed to add to closed connections list")
// 	}
// 	if closedConnectionBytes != nil {
// 		err = json.Unmarshal(closedConnectionBytes, &allClosedConnectionsList)
// 		if err != nil {
// 			return common.NewError("addToAllClosedConnectionsList_failed", "Failed to add to closed connections list")
// 		}
// 	}
// 	allClosedConnectionsList = append(allClosedConnectionsList, closedConnection)
// 	buff, _ := json.Marshal(allClosedConnectionsList)
// 	sc.DB.PutNode(smartcontractstate.Key("all_closed_connections"), buff)
// 	return nil
// }

func (sc *StorageSmartContract) Execute(t *transaction.Transaction, funcName string, input []byte) (string, error) {

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

	// if funcName == "close_connection" {
	// 	resp, err := sc.CloseConnectionWithBlobber(t, input)
	// 	if err != nil {
	// 		return "", err
	// 	}
	// 	return resp, nil
	// }

	if funcName == "new_allocation_request" {

		allBlobbersList, err := sc.getBlobbersList()
		if err != nil {
			return "", common.NewError("allocation_creation_failed", "No Blobbers registered. Failed to create a storage allocation")
		}

		if len(allBlobbersList) == 0 {
			return "", common.NewError("allocation_creation_failed", "No Blobbers registered. Failed to create a storage allocation")
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

				allocationBlobberKey := generateAllocationBlobberKey(t.Hash, blobberNode.ID)

				var blobberAllocation BlobberAllocation
				blobberAllocation.ID = allocationBlobberKey
				blobberAllocation.Size = (allocationRequest.Size + int64(size-1)) / int64(size)
				blobberAllocation.UsedSize = 0
				blobberAllocation.AllocationRoot = encryption.EmptyHash
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

	} else if funcName == "add_blobber" {
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

	return "", common.NewError("invalid_storage_function_name", "Invalid storage function called")
}

// func (sa *StorageAllocation) ProcessTransaction(t *transaction.Transaction) string {
// 	if sa.NumReads > 0 && sa.NumWrites > 0 && sa.Size > 0 && sa.DataShards > 0 {
// 		sa.Blobbers = mc.Blobbers.GetRandomNodes(sa.DataShards + sa.ParityShards)
// 		buff, _ := json.Marshal(sa)
// 		return string(buff)
// 	}
// 	return ""
// }
