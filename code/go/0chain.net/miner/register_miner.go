package miner

//register_miner client side
import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"time"

	"0chain.net/chaincore/chain"
	"0chain.net/chaincore/client"
	"0chain.net/chaincore/httpclientutil"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/wallet"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	. "0chain.net/core/logging"
	"0chain.net/core/memorystore"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

//Note: MinerNode is originally defined in MinerSmartcontract.
const (
	//MinerSCAddress address of minersc
	MinerSCAddress     = "CF9C03CD22C9C7B116EED04E4A909F95ABEC17E98FE631D6AC94D5D8420C5B20"
	getNodepoolInfoAPI = "/getNodepool"
)

const successConsesus = 33

//MinerNode struct that holds information about the registering miner
type MinerNode struct {
	ID        string `json:"id"`
	BaseURL   string `json:"url"`
	PublicKey string `json:"-"`
}

func (mn *MinerNode) encode() []byte {
	buff, _ := json.Marshal(mn)
	return buff
}

func (mn *MinerNode) decode(input []byte) error {
	err := json.Unmarshal(input, mn)
	if err != nil {
		return err
	}
	return nil
}

// PoolMembers Pool members of the blockchain
type PoolMembers struct {
	Miners   []string `json:"miners"`
	Sharders []string `json:"sharders"`
}

// PoolMemberInfo of pool members
type PoolMemberInfo struct {
	N2NHost   string `json:"n2n_host"`
	PublicKey string `json:"public_key"`
	Port      string `json:"port"`
	Type      string `json:"type"`
}

type PoolMembersInfo struct {
	MembersInfo []PoolMemberInfo `json:"members_info"`
}

const numRetriesForTxn = 3
const numRetriesForTxnConfirmation = 3
const scNameAddMiner = "add_miner"
const scNameViewchangeReq = "viewchange_req"
const scNameSyncReq = "sync_req"
const discoverIPPath = "/_nh/getpoolmembers"

var discoveryIps []string

var members PoolMembers
var myWallet *wallet.Wallet

//DiscoverPoolMembers given the discover_ips file, reads ips from it and discovers pool members
func DiscoverPoolMembers(discoveryFile string) bool {

	extractDiscoverIps(discoveryFile)

	var pm PoolMembers
	for _, ip := range discoveryIps {
		pm = PoolMembers{}

		httpclientutil.MakeGetRequest(ip+discoverIPPath, &pm)

		if pm.Miners != nil {
			if len(pm.Miners) == 0 {
				Logger.Info("Length of miners is 0")
			} else {
				sort.Strings(pm.Miners)
				sort.Strings(pm.Sharders)
				if len(members.Miners) == 0 {
					members = pm
					/*
						Logger.Info("First set of members from", zap.String("URL", ip),
							zap.Any("Miners", members.Miners), zap.Any("Sharders", members.Sharders))
					*/
				} else {
					if !isSliceEq(pm.Miners, members.Miners) || !isSliceEq(pm.Sharders, members.Sharders) {
						Logger.Info("The members are different from", zap.String("URL", ip),
							zap.Any("Miners", members.Miners), zap.Any("Sharders", pm.Sharders))
						return false
					}
				}

			}
		} else {
			Logger.Info("Miners are nil")
			return false
		}
	}
	if len(pm.Miners) > 0 {
		//Logger.Info("Discovered pool members", zap.Any("Miners", pm.Miners), zap.Any("Sharders", pm.Sharders))
		return true
	}

	Logger.Info("Could not discover Blockchain")
	return false

}

func extractDiscoverIps(discFile string) {
	//Logger.Info("The disc file", zap.String("name", discFile))
	ipsConfig := ReadYamlConfig(discFile)
	discIps := ipsConfig.Get("ips")

	if ips, ok := discIps.([]interface{}); ok {
		for _, nci := range ips {
			url, ok := nci.(map[interface{}]interface{})
			if !ok {
				continue
			}
			discoveryIps = append(discoveryIps, url["ip"].(string))
		}
	} else {
		Logger.Info("Could not read ips", zap.String("name", discFile))
	}
}

func isSliceEq(a, b []string) bool {

	// If one is nil, the other must also be nil.
	if (a == nil) != (b == nil) {
		return false
	}

	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}

//RegisterClient registers client on BC
func RegisterClient(sigScheme encryption.SignatureScheme) {
	Logger.Info("Registering client ")
	wallet.SetupWallet()
	myWallet = &wallet.Wallet{}
	err := myWallet.SetSignatureScheme(sigScheme)
	if err != nil {
		panic(err)
	}
	clientMetadataProvider := datastore.GetEntityMetadata("client")
	ctx := memorystore.WithEntityConnection(common.GetRootContext(), clientMetadataProvider)
	defer memorystore.Close(ctx)
	ctx = datastore.WithAsyncChannel(ctx, client.ClientEntityChannel)
	err = myWallet.Register(ctx)
	if err != nil {
		panic(err)
	}

	nodeBytes, _ := json.Marshal(myWallet)
	//Logger.Info("Post body", zap.Any("publicKey", myWallet.PublicKey), zap.String("ID", myWallet.ClientID))
	for _, ip := range members.Miners {
		body, err := httpclientutil.SendPostRequest(ip+httpclientutil.RegisterClient, nodeBytes, "", "", nil)
		if err != nil {
			Logger.Error("error in register client", zap.Error(err), zap.Any("body", body))
		}
		time.Sleep(httpclientutil.SleepBetweenRetries * time.Second)
	}
	//Logger.Info("My Client Info", zap.Any("ClientId", myWallet.ClientID))

}

func sendRegisterMinerReq() (string, error) {

	txn := httpclientutil.NewTransactionEntity(node.Self.ID, chain.GetServerChain().ID, node.Self.PublicKey)

	mn := &MinerNode{}
	mn.ID = node.Self.GetKey()
	mn.BaseURL = node.Self.GetURLBase()

	scData := &httpclientutil.SmartContractTxnData{}
	scData.Name = scNameAddMiner
	scData.InputArgs = mn

	txn.ToClientID = MinerSCAddress
	txn.Value = 0
	txn.TransactionType = httpclientutil.TxnTypeSmartContract
	txnBytes, err := json.Marshal(scData)
	if err != nil {
		Logger.Error("Returning error", zap.Error(err))
		return "", err
	}
	txn.TransactionData = string(txnBytes)

	signer := func(hash string) (string, error) {
		return node.Self.Sign(hash)
	}

	err = txn.ComputeHashAndSign(signer)
	if err != nil {
		Logger.Info("Signing Failed during registering miner to the mining network", zap.Error(err))
		return "", err
	}
	Logger.Info("Adding miner to the blockchain.", zap.String("txn", txn.Hash))
	httpclientutil.SendTransaction(txn, members.Miners, node.Self.ID, node.Self.PublicKey)
	return txn.Hash, nil
}

func registerMiner() {
	for i := 0; i < numRetriesForTxn; i++ {
		Logger.Info("Registering miner ", zap.Int("Attempt#", i))
		regMinerTxn, err := sendRegisterMinerReq()
		if err != nil {
			Logger.Fatal("Error while registering", zap.Error(err))

		} else {
			regTxn := verifyTransaction(regMinerTxn)
			if regTxn != nil {
				Logger.Info("Registration success!!!", zap.String("txn", regTxn.Hash))
				return
			}
		}
	}
	Logger.Fatal("Could not register/verify")

}

func sendRequestViewchangeReq() (string, error) {

	txn := httpclientutil.NewTransactionEntity(node.Self.ID, chain.GetServerChain().ID, node.Self.PublicKey)

	mn := &MinerNode{}
	mn.ID = node.Self.GetKey()
	mn.BaseURL = node.Self.GetURLBase()

	scData := &httpclientutil.SmartContractTxnData{}
	scData.Name = scNameViewchangeReq
	scData.InputArgs = mn

	txn.ToClientID = MinerSCAddress
	txn.Value = 0
	txn.TransactionType = httpclientutil.TxnTypeSmartContract
	txnBytes, err := json.Marshal(scData)
	if err != nil {
		Logger.Error("Returning error", zap.Error(err))
		return "", err
	}
	txn.TransactionData = string(txnBytes)

	signer := func(hash string) (string, error) {
		return node.Self.Sign(hash)
	}

	err = txn.ComputeHashAndSign(signer)
	if err != nil {
		Logger.Info("Signing Failed during registering miner to the mining network", zap.Error(err))
		return "", err
	}
	Logger.Info("Adding miner to the blockchain.", zap.String("txn", txn.Hash))
	httpclientutil.SendTransaction(txn, members.Miners, node.Self.ID, node.Self.PublicKey)
	return txn.Hash, nil
}

func requestViewchange() {
	for i := 0; i < numRetriesForTxn; i++ {
		Logger.Info("Requesting viewchange ", zap.Int("Attempt#", i))
		vcrTxn, err := sendRequestViewchangeReq()
		if err != nil {
			Logger.Fatal("Error while viewchange request", zap.Error(err))

		} else {
			vcrTxnConf := verifyTransaction(vcrTxn)
			if vcrTxnConf != nil {
				Logger.Info("ViewChange Request success!!!", zap.String("txn", vcrTxnConf.Hash), zap.String("conf", vcrTxnConf.TransactionOutput))

				return
			}
		}
	}
	Logger.Fatal("Could not register/verify")

}

func getNodepoolInfo() {
	params := make(map[string]string)
	params["baseurl"] = node.Self.GetURLBase()
	params["id"] = node.Self.ID
	var membersInfo PoolMembersInfo
	err := httpclientutil.MakeSCRestAPICall(MinerSCAddress, getNodepoolInfoAPI, params, members.Sharders, &membersInfo, successConsesus)
	if err != nil {
		Logger.Info("Err from MakeSCRestAPICall", zap.Error(err))
	}
	Logger.Info("OP from MakeSCRestAPICall", zap.Any("membersInfo", membersInfo))

}

//KickoffMinerRegistration kicks off a new miner registration process
func KickoffMinerRegistration(discoveryIps *string, signatureScheme encryption.SignatureScheme) {
	if discoveryIps != nil {
		Logger.Info("discovring blockchain")
		if !DiscoverPoolMembers(*discoveryIps) {
			Logger.Fatal("Cannot discover pool members")
		}
		RegisterClient(signatureScheme)
		registerMiner()
		getNodepoolInfo()
		requestViewchange()

	} else {
		Logger.Fatal("Discovery URLs are nil. Cannot discovery pool members")
	}
}

func verifyTransaction(regMinerTxn string) *httpclientutil.Transaction {

	for i := 0; i < numRetriesForTxnConfirmation; i++ {
		time.Sleep(httpclientutil.SleepBetweenRetries * time.Second)
		regTxn, err := httpclientutil.GetTransactionStatus(regMinerTxn, members.Sharders, successConsesus)
		if err == nil {
			return regTxn
		}

		Logger.Info("Could not get confirmation for registration request. Retrying... ", zap.Error(err))
	}
	return nil
}

//ReadYamlConfig read an yaml file
func ReadYamlConfig(file string) *viper.Viper {
	dir, fileName := filepath.Split(file)
	ext := filepath.Ext(fileName)
	if ext == "" {
		ext = ".yaml"
	} else {
		fileName = fileName[:len(fileName)-len(ext)]
	}
	format := ext[1:]
	if dir == "" {
		dir = "."
	} else if dir[0] != '.' {
		dir = "." + string(filepath.Separator) + dir
	}
	v := viper.New()
	v.AddConfigPath(dir)
	v.SetConfigName(fileName)
	v.SetConfigType(format)
	err := v.ReadInConfig()
	if err != nil {
		panic(fmt.Sprintf("error reading config file %v - %v\n", file, err))
	}
	return v
}
