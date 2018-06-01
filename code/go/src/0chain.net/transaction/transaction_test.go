package transaction

import (
	"context"
	"fmt"
	"testing"
	"time"

	"0chain.net/client"
	"0chain.net/common"
	"0chain.net/config"
	"0chain.net/datastore"
	"0chain.net/encryption"
	"0chain.net/memorystore"
	"0chain.net/node"
)

var keyPairs = make(map[string]string)
var publicKeys = make([]string, 0, 1000)

func BenchmarkTransactionVerify(b *testing.B) {
	common.SetupRootContext(node.GetNodeContext())
	client.SetupEntity()
	SetupEntity()

	publicKey, privateKey := encryption.GenerateKeys()
	keyPairs[publicKey] = privateKey
	publicKeys = append(publicKeys, publicKey)

	txnData := fmt.Sprintf("Txn: Pay %v from %s\n", 42, publicKey)
	t := datastore.GetEntityMetadata("txn").Instance().(*Transaction)
	t.ClientID = datastore.ToKey(encryption.Hash(publicKey))
	t.TransactionData = txnData
	t.CreationDate = common.Now()
	c := &client.Client{}
	c.PublicKey = publicKey
	c.ID = datastore.ToKey(encryption.Hash(publicKey))
	_, err := t.Sign(c, privateKey)
	if err != nil {
		fmt.Printf("Error signing\n")
	}
	ctx := common.GetRootContext()
	for i := 0; i < b.N; i++ {
		t.PublicKey = publicKey
		t.VerifySignature(ctx)
	}
}

func BenchmarkTransactionRead(b *testing.B) {
	common.SetupRootContext(node.GetNodeContext())
	client.SetupEntity()
	SetupEntity()

	ctx := memorystore.WithEntityConnection(context.Background(), transactionEntityMetadata)
	defer memorystore.Close(ctx)

	txn := transactionEntityMetadata.Instance().(*Transaction)
	txn.ChainID = config.GetMainChainID()
	txnIDs := make([]datastore.Key, 0, memorystore.BATCH_SIZE)
	getTxnsFunc := func(ctx context.Context, qe memorystore.CollectionEntity) bool {
		txnIDs = append(txnIDs, qe.GetKey())
		return len(txnIDs) != memorystore.BATCH_SIZE
	}

	memorystore.IterateCollection(ctx, txn.GetCollectionName(), getTxnsFunc, transactionEntityMetadata)
	txns, err := memorystore.AllocateEntities(memorystore.BATCH_SIZE, transactionEntityMetadata)
	if err != nil {
		fmt.Printf("Error allocating entities\n")
	}
	for i := 0; i < b.N; i++ {
		memorystore.MultiRead(ctx, transactionEntityMetadata, txnIDs, txns)
	}
}

func B1enchmarkTransactionWrite(t *testing.B) {
	common.SetupRootContext(node.GetNodeContext())
	client.SetupEntity()
	SetupEntity()
	fmt.Printf("time : %v\n", time.Now().UnixNano()/int64(time.Millisecond))
	numClients := 10
	createClients(numClients)
	start := time.Now()
	numTxns := 1000
	done := make(chan bool, numTxns)
	txnchannel := make(chan *Transaction, 10000)
	for i := 1; i <= 100; i++ {
		go processWorker(txnchannel, done)
	}
	for i := 0; i < numTxns; i++ {
		publicKey := publicKeys[i%numClients]
		pvtKey := keyPairs[publicKey]
		txnData := fmt.Sprintf("Txn(%v) Pay %v from %s\n", i, i%100, publicKey)
		postTransaction(pvtKey, publicKey, txnData, txnchannel, done)
	}
	for count := 0; true; {
		<-done
		count++
		if count == numTxns {
			close(txnchannel)
			break
		}
	}
	fmt.Printf("Elapsed time for txns: %v\n", time.Since(start))
	time.Sleep(10 * time.Second)
}

func createClients(numClients int) {
	start := time.Now()
	fmt.Printf("Testing at %v\n", start)
	done := make(chan bool, numClients)
	for i := 1; i <= numClients; i++ {
		publicKey, privateKey := encryption.GenerateKeys()
		keyPairs[publicKey] = privateKey
		publicKeys = append(publicKeys, publicKey)
		go postClient(privateKey, publicKey, done)
	}
	for count := 0; true; {
		<-done
		count++
		if count == numClients {
			break
		}
	}
	fmt.Printf("Elapsed time for clients: %v\n", time.Since(start))
	time.Sleep(time.Second)
}

func postClient(privateKey string, publicKey string, done chan<- bool) {
	entity := client.Provider()
	c, ok := entity.(*client.Client)
	if !ok {
		fmt.Printf("it's not ok!\n")
	}
	c.PublicKey = publicKey
	c.ID = datastore.ToKey(encryption.Hash(publicKey))
	ctx := datastore.WithAsyncChannel(context.Background(), client.ClientEntityChannel)
	//ctx := memorystore.WithEntityConnection(context.Background(),entity.GetEntityMetadata())
	_, err := client.PutClient(ctx, entity)
	//memorystore.Close(ctx)
	if err != nil {
		fmt.Printf("error for %v : %v\n", publicKey, err)
	}
	done <- true
}

func postTransaction(privateKey string, publicKey string, txnData string, txnChannel chan<- *Transaction, done chan<- bool) {
	entity := Provider()
	t, ok := entity.(*Transaction)
	if !ok {
		fmt.Printf("it's not ok!\n")
	}
	t.ClientID = datastore.ToKey(encryption.Hash(publicKey))
	t.TransactionData = txnData
	t.CreationDate = common.Now()
	c := &client.Client{}
	c.PublicKey = publicKey
	c.ID = datastore.ToKey(encryption.Hash(publicKey))
	signature, err := t.Sign(c, privateKey)
	encryption.Sign(privateKey, t.Hash)
	if err != nil {
		fmt.Printf("error signing %v\n", err)
		return
	}
	t.Signature = signature
	txnChannel <- t
}

func processWorker(txnChannel <-chan *Transaction, done chan<- bool) {
	ctx := memorystore.WithEntityConnection(context.Background(), transactionEntityMetadata)
	defer memorystore.Close(ctx)

	for entity := range txnChannel {
		ctx = datastore.WithAsyncChannel(ctx, TransactionEntityChannel)
		_, err := PutTransaction(ctx, entity)
		if err != nil {
			fmt.Printf("error for %v : %v\n", entity, err)
		}
		done <- true
	}
}
