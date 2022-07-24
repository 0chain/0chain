package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"time"

	"0chain.net/chaincore/currency"

	"go.uber.org/zap"

	"0chain.net/chaincore/httpclientutil"
	mptwallet "0chain.net/chaincore/wallet"
	"0chain.net/core/common"
	"0chain.net/core/encryption"
	. "0chain.net/core/logging"
	"0chain.net/core/viper"
)

const (
	// If posting a transaction fails or if it doesn't get confirmed, try
	// posting it again some number of times.
	executeRetries = 3

	// Regularly check if a posted transaction has made it into a block, up to a
	// certain timeout.
	confirmationLag     = 1  // seconds
	confirmationRetries = 15 // repetitions
	confirmationQuorum  = 33 // percentage

	airdropSize = 10000000000
	feeSize     = 10

	discoverIPPath = "/_nh/getpoolmembers"

	chainID = "0afc093ffb509f059c55478bc1a60351cef7b4e9c008a53a6cc8241ca8617dfe"
)

// All of the miners and sharders in the blockchain.
type poolMembers struct {
	Miners   []string `json:"Miners"`
	Sharders []string `json:"Sharders"`
}

var members poolMembers

// Given the discover_pool file, read the IP addresses in it to find our miners
// and sharders.
func discoverPoolMembers(discoveryFile string) {
	Logger.Info("Discovering blockchain")

	discoveryIps := extractDiscoverIps(discoveryFile)

	var pm poolMembers
	for _, ip := range discoveryIps {
		pm = poolMembers{}

		_ = httpclientutil.MakeGetRequest(ip+discoverIPPath, &pm)

		if pm.Miners == nil {
			Logger.Info("Miners are nil")
			Logger.Fatal("Cannot discover pool members")
		}

		if len(pm.Miners) == 0 {
			Logger.Info("Length of Miners is 0")
			continue
		}

		sort.Strings(pm.Miners)
		sort.Strings(pm.Sharders)

		if len(members.Miners) == 0 {
			members = pm
			// Logger.Info("First set of members from", zap.String("URL", ip),
			//		zap.Any("Miners", members.Miners), zap.Any("Sharders", members.Sharders))
		} else {
			if !isSliceEq(pm.Miners, members.Miners) || !isSliceEq(pm.Sharders, members.Sharders) {
				Logger.Fatal("The members are different from", zap.String("URL", ip),
					zap.Any("Miners", members.Miners), zap.Any("Sharders", pm.Sharders))
			}
		}
	}

	if len(pm.Miners) == 0 {
		Logger.Fatal("Could not discover blockchain")
	}

	Logger.Info("Discovered pool members", zap.Any("Miners", pm.Miners), zap.Any("Sharders", pm.Sharders))
}

func extractDiscoverIps(discFile string) []string {
	ipsConfig := readYamlConfig(discFile)
	discIps := ipsConfig.Get("ips")

	var discoveryIps []string

	if ips, ok := discIps.([]interface{}); ok {
		for _, nci := range ips {
			url, ok := nci.(map[interface{}]interface{})
			if !ok {
				continue
			}
			discoveryIps = append(discoveryIps, url["ip"].(string))
		}
	} else {
		Logger.Fatal("Could not read discovery file", zap.String("name", discFile))
	}

	return discoveryIps
}

// Read a yaml file from disk.
func readYamlConfig(file string) *viper.Viper {
	v := viper.New()
	if err := v.ReadConfigFile(file); err != nil {
		panic(fmt.Sprintf("Error reading config file %v - %v\n", file, err))
	}

	return v
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

// Get the chain's owner wallet. It can mint tokens, which we will use to
// populate our multi-sig wallets and their signers' wallets so we can actually
// use them.
func getOwnerWallet(signatureScheme, ownerKeysFile string) mptwallet.Wallet {
	reader, err := os.Open(ownerKeysFile)
	if err != nil {
		panic(err)
	}

	scheme := encryption.GetSignatureScheme(signatureScheme)

	err = scheme.ReadKeys(reader)
	if err != nil {
		panic(err)
	}

	w := mptwallet.Wallet{}

	err = w.SetSignatureScheme(scheme)
	if err != nil {
		panic(err)
	}

	return w
}

// Register a client on the blockchain's MPT.
func registerMPTWallet(w mptwallet.Wallet) {
	Logger.Info("Registering MPT wallet", zap.Any("ClientID", w.ClientID))

	data, err := json.Marshal(w)
	if err != nil {
		panic(err)
	}

	for _, ip := range members.Miners {
		body, err := httpclientutil.SendPostRequest(ip+httpclientutil.RegisterClient, data, "", "", nil)
		if err != nil {
			Logger.Fatal("HTTP POST error", zap.Error(err), zap.Any("body", body))
		}
	}

	Logger.Info("Success on registering MPT wallet")
}

func executeSCTransaction(from mptwallet.Wallet, scAddress string, value int64, data interface{}) httpclientutil.Transaction {
	dataBytes, err := json.Marshal(&data)
	if err != nil {
		Logger.Fatal("Failed to marshal data", zap.Error(err))
	}

	return executeTransaction(from, scAddress, value, httpclientutil.TxnTypeSmartContract, string(dataBytes))
}

func airdrop(owner mptwallet.Wallet, recipientClientID string) {
	Logger.Info("Requesting airdrop for MPT wallet", zap.String("ClientID", recipientClientID))
	executeTransaction(owner, recipientClientID, airdropSize, httpclientutil.TxnTypeSend, "Airdrop")
}

func executeTransaction(from mptwallet.Wallet, toClientID string, value int64, txnType int, data string) httpclientutil.Transaction {
	var err error

	for i := 0; i < executeRetries; i++ {
		hash := postTransaction(from, toClientID, value, from.Nonce+1, txnType, data)

		t, err := confirmTransaction(hash)
		if err == nil {
			from.Nonce += 1
			return t
		}

		if i != executeRetries-1 {
			Logger.Info("Transaction not found on sharders, retrying...", zap.Int("retry#", i+1), zap.Error(err))
		}
	}

	Logger.Fatal("Submitting transaction failed too many times", zap.Error(err))
	return httpclientutil.Transaction{} // Never reached.
}

func postTransaction(from mptwallet.Wallet, toClientID string, value int64, nonce int64, txnType int, data string) string {
	txn := httpclientutil.Transaction{
		ClientID:  from.ClientID,
		PublicKey: from.PublicKey,

		ToClientID:      toClientID,
		ChainID:         chainID,
		TransactionData: data,
		Value:           value,
		CreationDate:    common.Now(),
		Fee:             feeSize,
		Nonce:           nonce,

		TransactionType: txnType,
	}
	txn.Version = "1.0"

	signer := func(hash string) (string, error) {
		return from.SignatureScheme.Sign(hash)
	}

	err := txn.ComputeHashAndSign(signer)
	if err != nil {
		Logger.Fatal("Could not sign transaction with public key", zap.Error(err))
	}

	httpclientutil.SendTransaction(&txn, members.Miners, "", "")

	return txn.Hash
}

func confirmTransaction(hash string) (httpclientutil.Transaction, error) {
	var e error

	for i := 0; i < confirmationRetries; i++ {
		time.Sleep(confirmationLag * time.Second)

		t, err := httpclientutil.GetTransactionStatus(hash, members.Sharders, confirmationQuorum)
		if err == nil {
			return *t, nil
		}

		e = err
	}

	return httpclientutil.Transaction{}, e
}

func getBalance(clientID string) currency.Coin {
	ctx, cancel := context.WithTimeout(context.Background(), 7*time.Second)
	defer cancel()
	balance, err := httpclientutil.MakeClientBalanceRequest(ctx, clientID, members.Sharders, confirmationQuorum)
	if err != nil {
		Logger.Fatal("Couldn't get client balance", zap.Error(err))
	}

	return balance
}

func clientIDForKey(key encryption.SignatureScheme) string {
	publicKeyBytes, err := hex.DecodeString(key.GetPublicKey())
	if err != nil {
		panic(err)
	}

	return encryption.Hash(publicKeyBytes)
}
