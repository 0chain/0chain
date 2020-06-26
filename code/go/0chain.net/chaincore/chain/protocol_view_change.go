package chain

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"

	"0chain.net/chaincore/client"
	"0chain.net/chaincore/config"
	"0chain.net/chaincore/httpclientutil"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	. "0chain.net/core/logging"
	"0chain.net/core/memorystore"
	"0chain.net/core/util"
	"0chain.net/smartcontract/minersc"

	"github.com/spf13/viper"
	"go.uber.org/zap"
	// temporary debug
	// "runtime"
)

const (
	scNameAddMiner    = "add_miner"
	scNameAddSharder  = "add_sharder"
	scNameSharderKeep = "sharder_keep"
)
const (
	scRestAPIGetMinerList       = "/getMinerList"
	scRestAPIGetSharderList     = "/getSharderList"
	scRestAPIGetSharderKeepList = "/getSharderKeepList"
)

func (mc *Chain) InitSetupSC() {
	registered := mc.isRegistered()
	for !registered {
		txn, err := mc.RegisterNode()
		if err != nil {
			Logger.Warn("failed to register node in SC -- init_setup_sc", zap.Error(err))
		} else if !mc.ConfirmTransaction(txn) {
			time.Sleep(time.Second)
		}
		registered = mc.isRegistered()
	}
}

//RegisterClient registers client on BC
func (mc *Chain) RegisterClient() {
	thresholdByCount := config.GetThresholdCount()
	if node.Self.Underlying().Type == node.NodeTypeMiner {
		clientMetadataProvider := datastore.GetEntityMetadata("client")
		ctx := memorystore.WithEntityConnection(common.GetRootContext(), clientMetadataProvider)
		defer memorystore.Close(ctx)
		ctx = datastore.WithAsyncChannel(ctx, client.ClientEntityChannel)
		_, err := client.PutClient(ctx, &node.Self.Underlying().Client)
		if err != nil {
			panic(err)
		}
	}

	mb := mc.GetCurrentMagicBlock()
	nodeBytes, _ := json.Marshal(node.Self.Underlying().Client)
	miners := mb.Miners.CopyNodesMap()
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

func (mc *Chain) isRegistered() (is bool) {
	is = mc.isRegisteredEx(
		func(n *node.Node) util.Path {
			if typ := n.Type; typ == node.NodeTypeMiner {
				return util.Path(encryption.Hash(minersc.AllMinersKey))
			} else if typ == node.NodeTypeSharder {
				return util.Path(encryption.Hash(minersc.AllShardersKey))
			}
			return nil
		},
		func(n *node.Node) string {
			if typ := n.Type; typ == node.NodeTypeMiner {
				return scRestAPIGetMinerList
			} else if typ == node.NodeTypeSharder {
				return scRestAPIGetSharderList
			}
			return ""
		})
	return
}

func (mc *Chain) isRegisteredEx(getStatePath func(n *node.Node) util.Path,
	getAPIPath func(n *node.Node) string) bool {
	allMinersList := &minersc.MinerNodes{}
	currentNode := node.Self.Underlying()
	if mc.ActiveInChain() {
		clientState := CreateTxnMPT(mc.GetLatestFinalizedBlock().ClientState)
		statePath := getStatePath(currentNode)
		nodeList, err := clientState.GetNodeValue(statePath)
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
		mb := mc.GetCurrentMagicBlock()
		var (
			sharders = mb.Sharders.N2NURLs()
			err      error
		)
		relPath := getAPIPath(currentNode)
		err = httpclientutil.MakeSCRestAPICall(minersc.ADDRESS, relPath, nil, sharders, allMinersList, 1)
		if err != nil {
			Logger.Error("is registered", zap.Any("error", err))
			return false
		}
	}

	for _, miner := range allMinersList.Nodes {
		if miner.ID == currentNode.GetKey() {
			return true
		}
	}
	return false
}

func (mc *Chain) ConfirmTransaction(t *httpclientutil.Transaction) bool {
	var (
		active = mc.ActiveInChain()
		mb     = mc.GetCurrentMagicBlock()

		found, pastTime bool
		urls            []string
	)

	for _, sharder := range mb.Sharders.CopyNodesMap() {
		if !active || sharder.GetStatus() == node.NodeStatusActive {
			urls = append(urls, sharder.GetN2NURLBase())
		}
	}

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
	selfNode := node.Self.Underlying()
	txn := httpclientutil.NewTransactionEntity(selfNode.GetKey(),
		mc.ID, selfNode.PublicKey)

	mn := minersc.NewMinerNode()
	mn.ID = selfNode.GetKey()
	mn.N2NHost = selfNode.N2NHost
	mn.Host = selfNode.Host
	mn.Port = selfNode.Port
	mn.PublicKey = selfNode.PublicKey
	mn.ShortName = selfNode.Description
	mn.BuildTag = selfNode.Info.BuildTag

	// miner SC configurations
	mn.DelegateWallet = viper.GetString("delegate_wallet")
	mn.ServiceCharge = viper.GetFloat64("service_charge")
	mn.NumberOfDelegates = viper.GetInt("number_of_delegates")
	mn.MinStake = state.Balance(viper.GetFloat64("min_stake") * 1e10)
	mn.MaxStake = state.Balance(viper.GetFloat64("max_stake") * 1e10)

	scData := &httpclientutil.SmartContractTxnData{}
	if selfNode.Type == node.NodeTypeMiner {
		scData.Name = scNameAddMiner
	} else if selfNode.Type == node.NodeTypeSharder {
		scData.Name = scNameAddSharder
	}

	scData.InputArgs = mn

	txn.ToClientID = minersc.ADDRESS
	txn.PublicKey = selfNode.PublicKey
	mb := mc.GetCurrentMagicBlock()
	var minerUrls = mb.Miners.N2NURLs()
	err := httpclientutil.SendSmartContractTxn(txn, minersc.ADDRESS, 0, 0, scData, minerUrls)
	return txn, err
}

func (mc *Chain) RegisterSharderKeep() (result *httpclientutil.Transaction, err2 error) {
	selfNode := node.Self.Underlying()
	if selfNode.Type != node.NodeTypeSharder {
		return nil, errors.New("only sharder")
	}
	txn := httpclientutil.NewTransactionEntity(selfNode.GetKey(),
		mc.ID, selfNode.PublicKey)

	mn := minersc.NewMinerNode()
	mn.ID = selfNode.GetKey()
	mn.N2NHost = selfNode.N2NHost
	mn.Host = selfNode.Host
	mn.Port = selfNode.Port
	mn.PublicKey = selfNode.PublicKey
	mn.ShortName = selfNode.Description
	mn.BuildTag = selfNode.Info.BuildTag

	scData := &httpclientutil.SmartContractTxnData{}
	scData.Name = scNameSharderKeep
	scData.InputArgs = mn

	txn.ToClientID = minersc.ADDRESS
	txn.PublicKey = selfNode.PublicKey
	mb := mc.GetCurrentMagicBlock()
	var minerUrls = mb.Miners.N2NURLs()
	err := httpclientutil.SendSmartContractTxn(txn, minersc.ADDRESS, 0, 0, scData, minerUrls)
	return txn, err
}

func (mc *Chain) IsRegisteredSharderKeep() bool {
	return mc.isRegisteredEx(
		func(n *node.Node) util.Path {
			if typ := n.Type; typ == node.NodeTypeSharder {
				return util.Path(encryption.Hash(minersc.ShardersKeepKey))
			}
			return nil
		},
		func(n *node.Node) string {
			if typ := n.Type; typ == node.NodeTypeSharder {
				return scRestAPIGetSharderKeepList
			}
			return ""
		})
}
