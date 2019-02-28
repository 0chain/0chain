package minersc

import (
	"0chain.net/chaincore/smartcontractinterface"
	c_state "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"encoding/json"
	"go.uber.org/zap"
	. "0chain.net/core/logging"
)

const (
	//ADDRESS address of minersc
	ADDRESS = "CF9C03CD22C9C7B116EED04E4A909F95ABEC17E98FE631D6AC94D5D8420C5B20"
)

//MinerSmartContract Smartcontract that takes care of all miner related requests
type MinerSmartContract struct {
	*smartcontractinterface.SmartContract
}

//SetSC setting up smartcontract as per interface
func (msc *MinerSmartContract) SetSC(sc *smartcontractinterface.SmartContract) {
	msc.SmartContract = sc
}

func (msc *MinerSmartContract) getMinersList() ([]MinerNode, error) {
	var allMinersList = make([]MinerNode, 0)
	allMinersBytes, err := msc.DB.GetNode(allMinersKey)
	if err != nil {
		return nil, common.NewError("getMinersList_failed", "Failed to retrieve existing miners list")
	}
	if allMinersBytes == nil {
		return allMinersList, nil
	}
	err = json.Unmarshal(allMinersBytes, &allMinersList)
	if err != nil {
		return nil, common.NewError("getBlobbersList_failed", "Failed to retrieve existing blobbers list")
	}
	return allMinersList, nil
}

//AddMiner Function to handle miner register
func (msc *MinerSmartContract) AddMiner(t *transaction.Transaction, input []byte) (string, error) {
	

	allMinersList, err := msc.getMinersList()
	if err != nil {
		Logger.Error("Error in getting list from the DB", zap.Error(err))
		return "", common.NewError("add_miner_failed", "Failed to get miner list"+err.Error())
	}

	var newMiner MinerNode
	err = newMiner.decode(input) 
	if err != nil {
		Logger.Error("Error in decoding the input", zap.Error(err))
		
		return "", err
	}
	Logger.Info("The new miner info", zap.String("base URL", newMiner.BaseURL))
	newMiner.ID = t.ClientID
	newMiner.PublicKey = t.PublicKey
	minerBytes, _ := msc.DB.GetNode(newMiner.getKey())
	if minerBytes == nil {
		//DB does not have the miner already
		allMinersList = append(allMinersList, newMiner)
		allMinersBytes, _ := json.Marshal(allMinersList)
		msc.DB.PutNode(allMinersKey, allMinersBytes)
		msc.DB.PutNode(newMiner.getKey(), newMiner.encode())
		Logger.Info("Adding miner to known list of miners", zap.Any("url", allMinersList))
	}  else {
		Logger.Info("Miner received already exist", zap.String("url", newMiner.BaseURL))
	}

	buff := newMiner.encode()
	return string(buff), nil
}


//Execute implemetning the interface
func (msc *MinerSmartContract) Execute(t *transaction.Transaction, funcName string, input []byte, balances c_state.StateContextI) (string, error) {

	switch funcName {

		case "add_miner": 
			resp, err := msc.AddMiner(t, input)
			if err != nil {
				return "", err
			}
			return resp, nil
		
	default:
		return common.NewError("failed execution", "no function with that name").Error(), nil
	
	}
}
