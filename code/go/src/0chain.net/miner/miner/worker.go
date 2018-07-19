package main

import (
	"fmt"
	"math/rand"
	"time"

	"0chain.net/chain"

	"0chain.net/miner"

	"0chain.net/client"
	"0chain.net/common"
	"0chain.net/datastore"
	"0chain.net/memorystore"
	"0chain.net/transaction"
	"0chain.net/wallet"
)

var wallets []*wallet.Wallet

/*TransactionGenerator - generates a steady stream of transactions */
func TransactionGenerator(blockSize int32) {
	wallet.SetupWallet()
	GenerateClients(1024)
	csize := len(wallets)
	numTxns := blockSize
	P := time.Duration(1 + blockSize/1000)
	N := time.Duration(2)
	ticker := time.NewTicker(N*chain.DELTA + P*100*time.Millisecond)
	numWorkers := 1
	switch {
	case blockSize <= 10:
		numWorkers = 1
	case blockSize <= 100:
		numWorkers = 5
	case blockSize <= 1000:
		numWorkers = 10
	case blockSize <= 10000:
		numWorkers = 25
	case blockSize <= 100000:
		numWorkers = 50
	default:
		numWorkers = 100
	}
	txnMetadataProvider := datastore.GetEntityMetadata("txn")

	txnChannel := make(chan bool, blockSize)
	for i := 0; i < numWorkers; i++ {
		ctx := memorystore.WithEntityConnection(common.GetRootContext(), txnMetadataProvider)
		ctx = datastore.WithAsyncChannel(ctx, transaction.TransactionEntityChannel)
		go func() {
			rs := rand.NewSource(time.Now().UnixNano())
			prng := rand.New(rs)
			for range txnChannel {
				var wf, wt *wallet.Wallet
				for true {
					wf = wallets[prng.Intn(csize)]
					wt = wallets[prng.Intn(csize)]
					if wf != wt {
						break
					}
				}
				txn := wf.CreateTransaction(wt.ClientID)
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
	for true {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			txnCount := int32(txnMetadataProvider.GetStore().GetCollectionSize(ctx, txnMetadataProvider, collectionName))
			if txnCount >= 20*blockSize {
				continue
			}
			for i := int32(0); i < numTxns; i++ {
				txnChannel <- true
			}
		}
	}
}

/*GenerateClients - generate the given number of clients */
func GenerateClients(numClients int) {
	clientMetadataProvider := datastore.GetEntityMetadata("client")
	ctx := memorystore.WithEntityConnection(common.GetRootContext(), clientMetadataProvider)
	defer memorystore.Close(ctx)
	ctx = datastore.WithAsyncChannel(ctx, client.ClientEntityChannel)
	for i := 0; i < numClients; i++ {
		//client side code
		w := &wallet.Wallet{}
		w.Initialize()
		wallets = append(wallets, w)

		//Server side code bypassing REST for speed
		c := clientMetadataProvider.Instance().(*client.Client)
		c.PublicKey = w.PublicKey
		c.ID = w.ClientID
		_, err := client.PutClient(ctx, c)
		if err != nil {
			panic(err)
		}
	}
}
