package storagesc

import (
	"encoding/json"

	"0chain.net/common"
	"0chain.net/encryption"
	"0chain.net/smartcontractinterface"
	"0chain.net/smartcontractstate"
	"0chain.net/transaction"

	. "0chain.net/logging"
	"go.uber.org/zap"
)

const Seperator = smartcontractinterface.Seperator

const (
	Unknown = iota
	Active
	Closed
)

type StorageNode struct {
	ID        string `json:"id"`
	BaseURL   string `json:"url"`
	PublicKey string `json:"-"`
}

func (sn *StorageNode) Encode() []byte {
	buff, _ := json.Marshal(sn)
	return buff
}

func (sn *StorageNode) Decode(input []byte) error {
	err := json.Unmarshal(input, sn)
	if err != nil {
		return err
	}
	return nil
}

type StorageAllocation struct {
	ID           string                  `json:"id"`
	NumReads     int64                   `json:"num_reads"`
	NumWrites    int64                   `json:"num_writes"`
	DataShards   int                     `json:"data_shards"`
	ParityShards int                     `json:"parity_shards"`
	Size         int64                   `json:"size"`
	Expiration   common.Timestamp        `json:"expiration_date"`
	Blobbers     map[string]*StorageNode `json:"blobbers"`
	Owner        string                  `json:"owner_id"`
}

type BlobberAllocation struct {
	ID                   string       `json:"id"`
	Size                 int64        `json:"size"`
	Capacity             int64        `json:"capacity"`
	AllocationMerkleRoot string       `json:"allocation_merkle_root"`
	RedeemedWriteCounter int64        `json:"latest_write_counter_redeemed"`
	BlobberNode          *StorageNode `json:"storage_node"`
}

type StorageConnection struct {
	ClientID      string `json:"client_id"`
	BlobberID     string `json:"blobber_id"`
	MaxSize       int64  `json:"max_size"`
	AllocationID  string `json:"allocation_id"`
	TransactionID string `json:"transaction_id"`
	Status        int    `json:"status"`
}

type StorageSmartContract struct {
	smartcontractinterface.SmartContract
}

var allBlobbersMap = make(map[string]*StorageNode)
var allBlobbersList = make([]string, 0)
var allocationRequestMap = make(map[string]*StorageAllocation)
var blobberAllocationMap = make(map[string]*BlobberAllocation)
var allOpenConnectionsMap = make(map[string]*StorageConnection)

func (sc *StorageSmartContract) newAllocationReqeust(inputData []byte) *StorageAllocation {
	var storageAllocation StorageAllocation
	err := json.Unmarshal(inputData, &storageAllocation)
	if err != nil {
		return nil
	}

	return &storageAllocation
}

func generateClientBlobberKey(allocationID string, clientID string, blobberID string) string {
	return encryption.Hash(allocationID + Seperator + clientID + Seperator + blobberID)
}

func (sc *StorageSmartContract) OpenConnectionWithBlobber(t *transaction.Transaction, input []byte) (string, error) {
	var openConnection StorageConnection
	err := json.Unmarshal(input, &openConnection)
	if err != nil {
		return "", err
	}
	if len(openConnection.ClientID) == 0 || len(openConnection.BlobberID) == 0 || len(openConnection.AllocationID) == 0 {
		return "", common.NewError("invalid_parameters", "Invalid ClientID, BlobberID or Allocation ID for opening connection.")
	}

	allocationObj, ok := allocationRequestMap[openConnection.AllocationID]
	if !ok {
		return "", common.NewError("invalid_parameters", "Invalid allocation ID")
	}

	if t.ClientID != openConnection.ClientID || allocationObj.Owner != t.ClientID {
		return "", common.NewError("invalid_parameters", "Connection has to be opened by the same client as owner of the allocation")
	}

	clientBlobberKey := generateClientBlobberKey(openConnection.AllocationID, openConnection.ClientID, openConnection.BlobberID)
	//check for existing open connections
	if _, ok := allOpenConnectionsMap[clientBlobberKey]; ok {
		Logger.Error("An open connection to the blobber already exists from this client", zap.Any("clientBlobberKey", clientBlobberKey), zap.Any("openConnection", allOpenConnectionsMap[clientBlobberKey]))
		return "", common.NewError("connection_to_blobber_exists", "An open connection to the blobber already exists from this client")
	}

	//check for blobber has enough capacity for the operation
	blobberAllocation := blobberAllocationMap[clientBlobberKey]
	if blobberAllocation.Capacity < openConnection.MaxSize {
		return "", common.NewError("insufficient_capacity", "The blobber does not have enough storage capacity to handle this request")
	}

	openConnection.ClientID = t.ClientID
	openConnection.TransactionID = t.Hash
	openConnection.Status = Active
	allOpenConnectionsMap[clientBlobberKey] = &openConnection

	buff, _ := json.Marshal(openConnection)
	return string(buff), nil
}

func (sc *StorageSmartContract) CloseConnectionWithBlobber(t *transaction.Transaction, input []byte) (string, error) {
	var openConnection StorageConnection
	err := json.Unmarshal(input, &openConnection)
	if err != nil {
		return "", err
	}

	if len(openConnection.ClientID) == 0 || len(openConnection.BlobberID) == 0 || len(openConnection.AllocationID) == 0 {
		return "", common.NewError("invalid_parameters", "Invalid ClientID, BlobberID or Allocation ID for closing connection.")
	}

	clientBlobberKey := generateClientBlobberKey(openConnection.AllocationID, openConnection.ClientID, openConnection.BlobberID)
	//check for existing open connections
	if _, ok := allOpenConnectionsMap[clientBlobberKey]; !ok {
		Logger.Error("An open connection to the blobber from this client could not be found", zap.Any("clientBlobberKey", clientBlobberKey), zap.Any("openConnection", allOpenConnectionsMap[clientBlobberKey]))
		return "", common.NewError("no_connection_to_blobber_exists", "An open connection to the blobber from this client could not be found")
	}

	storedOpenConnection := allOpenConnectionsMap[clientBlobberKey]
	if storedOpenConnection.Status != Active {
		return "", common.NewError("invalid_client", "Connection is not active. So cannot close the connection")
	}

	if storedOpenConnection.BlobberID != t.ClientID {
		return "", common.NewError("invalid_client", "Connection cannot be closed by anyone apart from the blobber")
	}

	if storedOpenConnection.AllocationID != openConnection.AllocationID || storedOpenConnection.TransactionID != openConnection.TransactionID || storedOpenConnection.BlobberID != openConnection.BlobberID || storedOpenConnection.ClientID != openConnection.ClientID {
		Logger.Error("An open connection to the blobber from this client could not be found", zap.Any("clientBlobberKey", clientBlobberKey), zap.Any("storedOpenConnection", storedOpenConnection), zap.Any("openConnection", openConnection))
		return "", common.NewError("invalid_input_data", "Invalid input data to close the connection")
	}
	//check for blobber has accepted more data
	if allOpenConnectionsMap[clientBlobberKey].MaxSize < openConnection.MaxSize {
		return "", common.NewError("invalid_size_used", "Blobber has accepted more bytes than asked for by client")
	}

	blobberAllocation := blobberAllocationMap[clientBlobberKey]
	blobberAllocation.Capacity = blobberAllocation.Capacity - openConnection.MaxSize
	storedOpenConnection.Status = Closed
	openConnection.Status = Closed

	delete(allOpenConnectionsMap, clientBlobberKey)

	buff, _ := json.Marshal(openConnection)
	return string(buff), nil
}

func (sc *StorageSmartContract) Execute(t *transaction.Transaction, funcName string, input []byte) (string, error) {
	if funcName == "open_connection" {
		resp, err := sc.OpenConnectionWithBlobber(t, input)
		if err != nil {
			return "", err
		}
		return resp, nil
	}

	if funcName == "close_connection" {
		resp, err := sc.CloseConnectionWithBlobber(t, input)
		if err != nil {
			return "", err
		}
		return resp, nil
	}

	if funcName == "new_allocation_request" {
		allocationRequest := sc.newAllocationReqeust(input)
		if allocationRequest == nil {
			return "", common.NewError("allocation_creation_failed", "Failed to create a storage allocation")
		}
		if len(allBlobbersList) == 0 {
			return "", common.NewError("allocation_creation_failed", "No Blobbers registered. Failed to create a storage allocation")
		}
		if allocationRequest.NumReads > 0 && allocationRequest.NumWrites > 0 && allocationRequest.Size > 0 && allocationRequest.DataShards > 0 {
			size := allocationRequest.DataShards + allocationRequest.ParityShards

			if len(allBlobbersList) < allocationRequest.DataShards+allocationRequest.ParityShards {
				size = len(allBlobbersList)
			}
			allocatedBlobbers := make(map[string]*StorageNode)
			for i := 0; i < size; i++ {
				blobberID := allBlobbersList[i]
				blobberNode := allBlobbersMap[blobberID]
				clientBlobberKey := generateClientBlobberKey(t.Hash, t.ClientID, blobberID)

				var blobberAllocation BlobberAllocation
				blobberAllocation.ID = clientBlobberKey
				blobberAllocation.Size = (allocationRequest.Size + int64(size-1)) / int64(size)
				blobberAllocation.Capacity = blobberAllocation.Size
				blobberAllocation.RedeemedWriteCounter = 0
				blobberAllocationMap[clientBlobberKey] = &blobberAllocation
				allocatedBlobbers[clientBlobberKey] = blobberNode
			}
			allocationRequest.Blobbers = allocatedBlobbers
			allocationRequest.ID = t.Hash
			allocationRequest.Owner = t.ClientID
			buff, _ := json.Marshal(allocationRequest)
			allocationRequestMap[t.Hash] = allocationRequest
			return string(buff), nil
		}
		return "", common.NewError("invalid_allocation_request", "Failed storage allocate")

	} else if funcName == "add_blobber" {
		var newBlobber StorageNode
		err := json.Unmarshal(input, &newBlobber)
		if err != nil {
			return "", err
		}
		newBlobber.ID = t.ClientID
		newBlobber.PublicKey = t.PublicKey
		if _, ok := allBlobbersMap[newBlobber.ID]; !ok {
			allBlobbersMap[newBlobber.ID] = &newBlobber
			allBlobbersList = append(allBlobbersList, newBlobber.ID)
		}
		sc.DB.PutNode(smartcontractstate.Key(newBlobber.ID), newBlobber.Encode())
		buff, _ := json.Marshal(newBlobber)
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
