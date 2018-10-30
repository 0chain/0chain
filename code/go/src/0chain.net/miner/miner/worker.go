package main

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	"0chain.net/chain"
	"0chain.net/encryption"
	"go.uber.org/zap"

	"0chain.net/miner"

	"0chain.net/client"
	"0chain.net/common"
	"0chain.net/datastore"
	. "0chain.net/logging"
	"0chain.net/memorystore"
	"0chain.net/transaction"
	"0chain.net/wallet"
)

var wallets []*wallet.Wallet

/*TransactionGenerator - generates a steady stream of transactions */
func TransactionGenerator(blockSize int32) {
	wallet.SetupWallet()
	var numClients int32 = 1024
	if blockSize*4 < numClients {
		numClients = 4 * blockSize
	}
	GenerateClients(numClients)
	numWorkers := 1
	numTxns := blockSize
	switch {
	case blockSize <= 10:
		numWorkers = 1
	case blockSize <= 100:
		numWorkers = 5
	case blockSize <= 1000:
		numWorkers = 10
		numTxns = blockSize / 2
	case blockSize <= 10000:
		numWorkers = 25
		numTxns = blockSize / 2
	case blockSize <= 100000:
		numWorkers = 50
		numTxns = blockSize / 2
	default:
		numWorkers = 100
	}
	txnMetadataProvider := datastore.GetEntityMetadata("txn")
	txnChannel := make(chan bool, numTxns)
	for i := 0; i < numWorkers; i++ {
		ctx := datastore.WithAsyncChannel(common.GetRootContext(), transaction.TransactionEntityChannel)
		go func() {
			ctx = memorystore.WithEntityConnection(ctx, txnMetadataProvider)
			rs := rand.NewSource(time.Now().UnixNano())
			prng := rand.New(rs)
			var txn *transaction.Transaction
			for range txnChannel {
				r := prng.Int63n(100)
				if r < 25 {
					txn = createSendTransaction(prng)
				} else {
					txn = createDataTransaction(prng)
				}
				_, err := transaction.PutTransaction(ctx, txn)
				if err != nil {
					fmt.Printf("error:%v: %v\n", time.Now(), err)
					//panic(err)
				}
			}
		}()
	}
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

	for true {
		numTxns = rand.Int31n(333)
		numGenerators := sc.NumGenerators
		numMiners := sc.Miners.Size()
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
			for i := int32(0); i < numTxns; i++ {
				txnChannel <- true
			}
		}
	}
}

func createSendTransaction(prng *rand.Rand) *transaction.Transaction {
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
func GetOwnerWallet(keysFile string) *wallet.Wallet {
	reader, err := os.Open(keysFile)
	if err != nil {
		panic(err)
	}
	sigScheme := encryption.NewED25519Scheme()
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
func GenerateClients(numClients int32) {
	ownerWallet := GetOwnerWallet("config/owner_keys.txt")
	rs := rand.NewSource(time.Now().UnixNano())
	prng := rand.New(rs)

	clientMetadataProvider := datastore.GetEntityMetadata("client")
	ctx := memorystore.WithEntityConnection(common.GetRootContext(), clientMetadataProvider)
	defer memorystore.Close(ctx)
	ctx = datastore.WithAsyncChannel(ctx, client.ClientEntityChannel)

	txnMetadataProvider := datastore.GetEntityMetadata("txn")
	tctx := memorystore.WithEntityConnection(common.GetRootContext(), txnMetadataProvider)
	tctx = datastore.WithAsyncChannel(ctx, transaction.TransactionEntityChannel)

	for i := int32(0); i < numClients; i++ {
		//client side code
		w := &wallet.Wallet{}
		w.Initialize()
		wallets = append(wallets, w)

		//Server side code bypassing REST for speed
		err := w.Register(ctx)
		if err != nil {
			panic(err)
		}
	}
	time.Sleep(time.Second)
	for _, w := range wallets {
		//generous airdrop in dev/test mode :)
		txn := ownerWallet.CreateSendTransaction(w.ClientID, prng.Int63n(10000)*10000000000, "generous air drop! :)")
		_, err := transaction.PutTransaction(tctx, txn)
		if err != nil {
			fmt.Printf("error:%v: %v\n", time.Now(), err)
			//panic(err)
		}
	}
	Logger.Info("generation of wallets complete", zap.Int("wallets", len(wallets)))
}
