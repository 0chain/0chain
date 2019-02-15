package main

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"sync"
	"time"

	"0chain.net/chaincore/chain"
	"github.com/spf13/viper"
	"go.uber.org/zap"

	"0chain.net/miner"

	"0chain.net/chaincore/client"
	"0chain.net/chaincore/config"
	"0chain.net/chaincore/transaction"
	"0chain.net/chaincore/wallet"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	. "0chain.net/core/logging"
	"0chain.net/core/memorystore"
)

var (
	wallets             []*wallet.Wallet
	txn_generation_rate int32
	pourPoint           int64
	pourAmount          int64
)

/*TransactionGenerator - generates a steady stream of transactions */
func TransactionGenerator(c *chain.Chain) {
	wallet.SetupWallet()

	viper.SetDefault("development.txn_generation.pour_point", 100)
	pourPoint = viper.GetInt64("development.txn_generation.pour_point")
	viper.SetDefault("development.txn_generation.pour_amount", 10000)
	pourAmount = viper.GetInt64("development.txn_generation.pour_amount")

	viper.SetDefault("development.txn_generation.wallets", 1000)
	var numClients = viper.GetInt("development.txn_generation.wallets")

	GenerateClients(c, numClients)
	numWorkers := 1
	blockSize := c.BlockSize
	viper.SetDefault("development.txn_generation.transactions", blockSize)
	numTxns := viper.GetInt32("development.txn_generation.transactions")
	SetTxnGenRate(numTxns)
	switch {
	case blockSize <= 10:
		numWorkers = 1
	case blockSize <= 100:
		numWorkers = 1
	case blockSize <= 1000:
		numWorkers = 2
		numTxns = blockSize / 2
	case blockSize <= 10000:
		numWorkers = 4
		numTxns = blockSize / 2
	case blockSize <= 100000:
		numWorkers = 8
		numTxns = blockSize / 2
	default:
		numWorkers = 16
	}
	txnMetadataProvider := datastore.GetEntityMetadata("txn")
	ctx := memorystore.WithEntityConnection(common.GetRootContext(), txnMetadataProvider)
	txn := txnMetadataProvider.Instance().(*transaction.Transaction)
	txn.ChainID = miner.GetMinerChain().ID
	collectionName := txn.GetCollectionName()
	sc := chain.GetServerChain()

	//Ensure the initial set of transactions succeed or become invalid
	txnCount := int32(txnMetadataProvider.GetStore().GetCollectionSize(ctx, txnMetadataProvider, collectionName))
	for txnCount > blockSize {
		time.Sleep(20 * time.Millisecond)
		txnCount = int32(txnMetadataProvider.GetStore().GetCollectionSize(ctx, txnMetadataProvider, collectionName))
	}

	numGenerators := sc.NumGenerators
	numMiners := sc.Miners.Size()
	for true {
		numTxns = GetTxnGenRate()
		numWorkerTxns := numTxns / int32(numWorkers)
		if numWorkerTxns*int32(numWorkers) < numTxns {
			numWorkerTxns++
		}
		blockRate := chain.SteadyStateFinalizationTimer.Rate1()
		if chain.SteadyStateFinalizationTimer.Count() < 250 && blockRate < 2 {
			blockRate = 2
		}
		totalBlocks := float64(numGenerators) * blockRate
		blocksPerMiner := totalBlocks / float64(numMiners)
		if blocksPerMiner < 1 {
			blocksPerMiner = 1
		}
		waitTime := time.Millisecond * time.Duration(1000./1.05/blocksPerMiner)
		timer := time.NewTimer(waitTime)
		if sc.CurrentRound%100 == 0 {
			Logger.Info("background transactions generation", zap.Duration("frequency", waitTime), zap.Float64("blocks", blocksPerMiner))
		}
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			txnCount := int32(txnMetadataProvider.GetStore().GetCollectionSize(ctx, txnMetadataProvider, collectionName))
			if float64(txnCount) >= blocksPerMiner*float64(8*numTxns) {
				continue
			}
			wg := sync.WaitGroup{}
			for i := 0; i < numWorkers; i++ {
				ctx := datastore.WithAsyncChannel(common.GetRootContext(), transaction.TransactionEntityChannel)
				wg.Add(1)
				go func() {
					ctx = memorystore.WithEntityConnection(ctx, txnMetadataProvider)
					defer memorystore.Close(ctx)
					rs := rand.NewSource(time.Now().UnixNano())
					prng := rand.New(rs)
					var txn *transaction.Transaction
					for t := int32(0); t <= numWorkerTxns; t++ {
						r := prng.Int63n(100)
						if r < 25 {
							txn = createSendTransaction(c, prng)
						} else {
							txn = createDataTransaction(prng)
						}
						_, err := transaction.PutTransaction(ctx, txn)
						if err != nil {
							fmt.Printf("error:%v: %v\n", time.Now(), err)
							//panic(err)
						}
					}
					wg.Done()
				}()
			}
			wg.Wait()
		}
	}
}

func createSendTransaction(c *chain.Chain, prng *rand.Rand) *transaction.Transaction {
	var wf, wt *wallet.Wallet
	csize := len(wallets)
	for true {
		wf = wallets[prng.Intn(csize)]
		wt = wallets[prng.Intn(csize)]
		if wf != wt {
			break
		}
	}
	txn := wf.CreateRandomSendTransaction(wt.ClientID)
	return txn
}

func createDataTransaction(prng *rand.Rand) *transaction.Transaction {
	csize := len(wallets)
	wf := wallets[prng.Intn(csize)]
	txn := wf.CreateRandomDataTransaction()
	return txn
}

/*GetOwnerWallet - get the owner wallet. Used to get the initial state get going */
func GetOwnerWallet(c *chain.Chain) *wallet.Wallet {
	var keysFile string
	if c.ClientSignatureScheme == "ed25519" {
		keysFile = "config/owner_keys.txt"
	} else {
		keysFile = "config/b0owner_keys.txt"
	}
	reader, err := os.Open(keysFile)
	if err != nil {
		panic(err)
	}
	sigScheme := c.GetSignatureScheme()
	err = sigScheme.ReadKeys(reader)
	if err != nil {
		panic(err)
	}
	w := &wallet.Wallet{}
	err = w.SetSignatureScheme(sigScheme)
	if err != nil {
		panic(err)
	}
	clientMetadataProvider := datastore.GetEntityMetadata("client")
	ctx := memorystore.WithEntityConnection(common.GetRootContext(), clientMetadataProvider)
	defer memorystore.Close(ctx)
	ctx = datastore.WithAsyncChannel(ctx, client.ClientEntityChannel)
	err = w.Register(ctx)
	if err != nil {
		panic(err)
	}
	return w
}

/*GenerateClients - generate the given number of clients */
func GenerateClients(c *chain.Chain, numClients int) {
	ownerWallet := GetOwnerWallet(c)
	rs := rand.NewSource(time.Now().UnixNano())
	prng := rand.New(rs)

	clientMetadataProvider := datastore.GetEntityMetadata("client")
	ctx := memorystore.WithEntityConnection(common.GetRootContext(), clientMetadataProvider)
	defer memorystore.Close(ctx)
	ctx = datastore.WithAsyncChannel(ctx, client.ClientEntityChannel)

	txnMetadataProvider := datastore.GetEntityMetadata("txn")
	tctx := memorystore.WithEntityConnection(common.GetRootContext(), txnMetadataProvider)
	tctx = datastore.WithAsyncChannel(ctx, transaction.TransactionEntityChannel)

	for i := 0; i < numClients; i++ {
		//client side code
		w := &wallet.Wallet{}
		w.Initialize(c.ClientSignatureScheme)
		wallets = append(wallets, w)

		//Server side code bypassing REST for speed
		err := w.Register(ctx)
		if err != nil {
			panic(err)
		}
	}
	time.Sleep(1 * time.Second)
	for _, w := range wallets {
		//generous airdrop in dev/test mode :)
		txn := ownerWallet.CreateSendTransaction(w.ClientID, prng.Int63n(100000)*10000000000, "generous air drop! :) debug")
		_, err := transaction.PutTransaction(tctx, txn)
		if err != nil {
			fmt.Printf("error:%v: %v\n", time.Now(), err)
			//panic(err)
		}
	}
	Logger.Info("generation of wallets complete", zap.Int("wallets", len(wallets)))
}

//SetTxnGenRate - the txn generation rate
func SetTxnGenRate(newRate int32) {
	txn_generation_rate = newRate
}

//GetTxnGenRate - the txn generation rate
func GetTxnGenRate() int32 {
	return txn_generation_rate
}
