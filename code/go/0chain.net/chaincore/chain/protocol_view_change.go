package chain

import (
	"encoding/json"
	"fmt"
	"math"
	"time"

	"0chain.net/chaincore/client"
	"0chain.net/chaincore/config"
	"0chain.net/chaincore/httpclientutil"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	. "0chain.net/core/logging"
	"0chain.net/core/memorystore"
	"0chain.net/core/util"
	"0chain.net/smartcontract/minersc"

	"go.uber.org/zap"
)

const (
	scNameAddMiner          = "add_miner"
	scRestAPIGetMinerList   = "/getMinerList"
	scNameAddSharder        = "add_sharder"
	scRestAPIGetSharderList = "/getSharderList"
)

func (mc *Chain) InitSetup() {
	mc.RegisterClient()
	registered := mc.isRegistered()
	for !registered {
		txn, err := mc.RegisterNode()
		if err == nil && mc.ConfirmTransaction(txn) {
			registered = true
		} else {
			time.Sleep(time.Second)
		}
	}
}

//RegisterClient registers client on BC
func (mc *Chain) RegisterClient() {
	thresholdByCount := config.GetThresholdCount()
	if node.Self.Type == node.NodeTypeMiner {
		clientMetadataProvider := datastore.GetEntityMetadata("client")
		ctx := memorystore.WithEntityConnection(common.GetRootContext(), clientMetadataProvider)
		defer memorystore.Close(ctx)
		ctx = datastore.WithAsyncChannel(ctx, client.ClientEntityChannel)
		_, err := client.PutClient(ctx, &node.Self.Client)
		if err != nil {
			panic(err)
		}
	}

	nodeBytes, _ := json.Marshal(node.Self.Client)
	miners := mc.Miners.CopyMap()
	registered := 0
	consensus := int(math.Ceil((float64(thresholdByCount) / 100) * float64(len(miners))))
	if consensus > len(miners) {
		Logger.DPanic(fmt.Sprintf("number of miners %d is not enough relative to the threshold parameter %d%%(%d)", len(miners), thresholdByCount, consensus))
	}
	for registered < consensus {
		for key, miner := range miners {
			body, err := httpclientutil.SendPostRequest(miner.GetN2NURLBase()+httpclientutil.RegisterClient, nodeBytes, "", "", nil)
			if err != nil {
				Logger.Error("error in register client", zap.Error(err), zap.Any("body", body))
			} else {
				delete(miners, key)
				registered++
			}
			time.Sleep(httpclientutil.SleepBetweenRetries * time.Millisecond)
		}
		time.Sleep(httpclientutil.SleepBetweenRetries * time.Millisecond)
	}
}

func (mc *Chain) isRegistered() bool {
	allMinersList := &minersc.MinerNodes{}
	if mc.ActiveInChain() {
		clientState := CreateTxnMPT(mc.GetLatestFinalizedBlock().ClientState)
		var nodeList util.Serializable
		var err error
		if node.Self.Type == node.NodeTypeMiner {
			nodeList, err = clientState.GetNodeValue(util.Path(encryption.Hash(minersc.AllMinersKey)))
		} else if node.Self.Type == node.NodeTypeSharder {
			nodeList, err = clientState.GetNodeValue(util.Path(encryption.Hash(minersc.AllShardersKey)))
		}
		if err != nil {
			Logger.Error("failed to get magic block", zap.Any("error", err))
			return false
		}
		if nodeList == nil {
			return false
		}
		err = allMinersList.Decode(nodeList.Encode())
		if err != nil {
			Logger.Error("failed to decode magic block", zap.Any("error", err))
			return false
		}
	} else {
		var (
			sharders = mc.Sharders.N2NURLs()
			err      error
		)
		if node.Self.Type == node.NodeTypeMiner {
			err = httpclientutil.MakeSCRestAPICall(minersc.ADDRESS, scRestAPIGetMinerList, nil, sharders, allMinersList, 1)
		} else if node.Self.Type == node.NodeTypeSharder {
			err = httpclientutil.MakeSCRestAPICall(minersc.ADDRESS, scRestAPIGetSharderList, nil, sharders, allMinersList, 1)
		}

		if err != nil {
			Logger.Error("is registered", zap.Any("error", err))
			return false
		}
	}
	var registered bool
	for _, miner := range allMinersList.Nodes {
		if miner.ID == node.Self.ID {
			registered = true
			break
		}
	}
	return registered
}

func (mc *Chain) ConfirmTransaction(t *httpclientutil.Transaction) bool {
	active := mc.ActiveInChain()
	var (
		found, pastTime bool
		urls            []string
	)
	mc.Sharders.ForEach(func(sharder *node.Node) {
		if !active || sharder.IsActive() {
			urls = append(urls, sharder.GetN2NURLBase())
		}
	})
	for !found && !pastTime {
		txn, err := httpclientutil.GetTransactionStatus(t.Hash, urls, 1)
		if active {
			lfb := mc.GetLatestFinalizedBlock()
			pastTime = lfb != nil && !common.WithinTime(int64(lfb.CreationDate), int64(t.CreationDate), transaction.TXN_TIME_TOLERANCE)
		} else {
			blockSummary, err := httpclientutil.GetBlockSummaryCall(urls, 1, false)
			if err != nil {
				Logger.Info("confirm transaction", zap.Any("confirmation", false))
				return false
			}
			pastTime = blockSummary != nil && !common.WithinTime(int64(blockSummary.CreationDate), int64(t.CreationDate), transaction.TXN_TIME_TOLERANCE)
		}
		found = err == nil && txn != nil
		if !found {
			time.Sleep(time.Second)
		}
	}
	return found

}

func (mc *Chain) RegisterNode() (*httpclientutil.Transaction, error) {
	txn := httpclientutil.NewTransactionEntity(node.Self.ID, mc.ID, node.Self.PublicKey)

	mn := minersc.NewMinerNode()
	mn.ID = node.Self.GetKey()
	mn.N2NHost = node.Self.N2NHost
	mn.Host = node.Self.Host
	mn.Port = node.Self.Port
	mn.PublicKey = node.Self.PublicKey
	mn.ShortName = node.Self.Description
	mn.Percentage = .5 // add to config
	mn.BuildTag = node.Self.Info.BuildTag

	scData := &httpclientutil.SmartContractTxnData{}
	if node.Self.Type == node.NodeTypeMiner {
		scData.Name = scNameAddMiner
	} else if node.Self.Type == node.NodeTypeSharder {
		scData.Name = scNameAddSharder
	}

	scData.InputArgs = mn

	txn.ToClientID = minersc.ADDRESS
	txn.PublicKey = node.Self.PublicKey
	var minerUrls = mc.Miners.N2NURLs()
	err := httpclientutil.SendSmartContractTxn(txn, minersc.ADDRESS, 0, 0, scData, minerUrls)
	return txn, err
}
