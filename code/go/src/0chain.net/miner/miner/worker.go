package main

import (
	"math/rand"
	"time"

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
	ctx := datastore.WithAsyncChannel(common.GetRootContext(), transaction.TransactionEntityChannel)
	txnMetadataProvider := datastore.GetEntityMetadata("txn")
	ctx = memorystore.WithEntityConnection(ctx, txnMetadataProvider)
	ticker := time.NewTicker(2 * time.Second)
	numTxns := 2 * blockSize
	GenerateClients(100)
	for true {
		select {
		case <-ctx.Done():
			return
		case _ = <-ticker.C:
			if len(wallets) < 10000 && rand.Intn(100) < 10 {
				GenerateClients(100)
			}
			for i := int32(0); i < numTxns; i++ {
				rs := rand.NewSource(time.Now().UnixNano())
				prng := rand.New(rs)
				var wf, wt *wallet.Wallet
				for true {
					wf = wallets[prng.Intn(len(wallets))]
					wt = wallets[prng.Intn(len(wallets))]
					if wf != wt {
						break
					}
				}
				txn := wf.CreateTransaction(wt.ClientID)
				datastore.DoAsync(ctx, txn)
				transaction.TransactionCount++
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

		//Server side code by passing REST for speed
		client := clientMetadataProvider.Instance().(*client.Client)
		client.PublicKey = w.PublicKey
		client.ID = w.ClientID
		datastore.DoAsync(ctx, client)
	}
}
