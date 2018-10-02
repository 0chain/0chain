package storagesc

import (
	"encoding/json"

	"0chain.net/common"
	"0chain.net/smartcontractinterface"
	"0chain.net/transaction"
)

type StorageNode struct {
	ID      string `json:"id"`
	BaseURL string `json:"url"`
}

type StorageAllocation struct {
	ID           string           `json:"id"`
	NumReads     int64            `json:"num_reads"`
	NumWrites    int64            `json:"num_writes"`
	DataShards   int              `json:"data_shards"`
	ParityShards int              `json:"parity_shards"`
	Size         int64            `json:"size"`
	Expiration   common.Timestamp `json:"expiration_date"`
	Blobbers     []*StorageNode   `json:"blobbers"`
}

type StorageSmartContract struct {
	sc smartcontractinterface.SmartContract
}

var allBlobbersMap = make(map[string]*StorageNode)
var allBlobbersList = make([]string, 0)
var allocationRequestMap = make(map[string]*StorageAllocation)

func (sc *StorageSmartContract) newAllocationReqeust(inputData []byte) *StorageAllocation {
	var storageAllocation StorageAllocation
	err := json.Unmarshal(inputData, &storageAllocation)
	if err != nil {
		return nil
	}

	return &storageAllocation
}

func (sc *StorageSmartContract) Execute(t *transaction.Transaction, funcName string, input []byte) (string, error) {
	if funcName == "new_allocation_request" {
		allocationRequest := sc.newAllocationReqeust(input)
		if allocationRequest == nil {
			return "", common.NewError("allocation_creation_failed", "Failed to create a storage allocation")
		}
		if len(allBlobbersList) == 0 {
			return "", common.NewError("allocation_creation_failed", "No Blobbers registered. Failed to create a storage allocation")
		}
		if allocationRequest.NumReads > 0 && allocationRequest.NumWrites > 0 && allocationRequest.Size > 0 && allocationRequest.DataShards > 0 {
			size := len(allBlobbersList) //allocationRequest.DataShards + allocationRequest.ParityShards
			shuffled := make([]*StorageNode, allocationRequest.DataShards+allocationRequest.ParityShards)
			for i := 0; i < len(shuffled); i++ {
				blobberID := allBlobbersList[i%size]
				shuffled[i] = allBlobbersMap[blobberID]
			}
			allocationRequest.Blobbers = shuffled
			allocationRequest.ID = t.Hash
			buff, _ := json.Marshal(allocationRequest)
			allocationRequestMap[t.Hash] = allocationRequest
			return string(buff), nil
		}
		return "", common.NewError("invalid_allocation_request", "Failed storage allocate")

	} else if funcName == "add_blobber" {
		var newBlobber StorageNode
		err := json.Unmarshal(input, &newBlobber)
		if err != nil {
			return "", nil
		}
		if _, ok := allBlobbersMap[newBlobber.ID]; !ok {
			allBlobbersMap[newBlobber.ID] = &newBlobber
			allBlobbersList = append(allBlobbersList, newBlobber.ID)
		}

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
