package miner

//register_miner client side
import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"time"

	"0chain.net/chaincore/client"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	. "0chain.net/core/logging"
	"0chain.net/core/memorystore"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"0chain.net/chaincore/wallet"
)

const txnSubmitURL = "v1/transaction/put"
const txnVerifyURL = "v1/transaction/get/confirmation?hash="
var registerClient = "/v1/client/put"

//TxnConfirmationTime time to wait before checking the status
const TxnConfirmationTime = 15

// PoolMembers Pool members of the blockchain
type PoolMembers struct {
	Miners   []string `json:"miners"`
	Sharders []string `json:"sharders"`
}

var discoverIPPath = "/_nh/getpoolmembers"
var discoveryIps []string

var members PoolMembers
var myWallet *wallet.Wallet

//DiscoverPoolMembers given the discover_ips file, reads ips from it and discovers pool members
func DiscoverPoolMembers(discoveryFile string) bool {

	extractDiscoverIps(discoveryFile)

	var pm PoolMembers
	for _, ip := range discoveryIps {
		pm = PoolMembers{}

		common.MakeGetRequest(ip+discoverIPPath, &pm) 

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

//RegisterClient registers client only locally
func RegisterClient(sigScheme encryption.SignatureScheme) {
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
		body, err := common.SendPostRequest(ip + registerClient, nodeBytes, "", "", nil)
		if err!= nil {
			Logger.Error("error in register client", zap.Error(err), zap.Any("body", body) )
		} 
		time.Sleep(common.SleepBetweenRetries * time.Second)
	}
	//Logger.Info("My Client Info", zap.Any("ClientId", myWallet.ClientID))
	
}

//KickoffMinerRegistration kicks off a new miner registration process
func KickoffMinerRegistration(discoveryIps *string, signatureScheme encryption.SignatureScheme) {
	if discoveryIps != nil {
		Logger.Info("discovring blockchain")
		if !DiscoverPoolMembers(*discoveryIps) {
			Logger.Fatal("Cannot discover pool members")
		}
		RegisterClient(signatureScheme)
		
	} else {
		Logger.Fatal("Discovery URLs are nil. Cannot discovery pool members")
	}
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