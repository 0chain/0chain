package transaction

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"0chain.net/chaincore/client"
	"0chain.net/chaincore/config"
	"0chain.net/chaincore/node"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/core/memorystore"

	"github.com/stretchr/testify/require"
)

var keyPairs = make(map[string]string)
var publicKeys = make([]string, 0, 1000)

var sigSchemes = make([]encryption.SignatureScheme, 0, 1000)

var clientSignatureScheme = "bls0chain"

func init() {
	client.SetClientSignatureScheme(clientSignatureScheme)
}

func BenchmarkTransactionVerify(b *testing.B) {
	common.SetupRootContext(node.GetNodeContext())
	client.SetupEntity(memorystore.GetStorageProvider())
	SetupEntity(memorystore.GetStorageProvider())

	sigScheme := encryption.GetSignatureScheme(clientSignatureScheme)
	err := sigScheme.GenerateKeys()
	if err != nil {
		panic(err)
	}
	sigSchemes = append(sigSchemes, sigScheme)

	c := &client.Client{}
	c.SetPublicKey(sigScheme.GetPublicKey())

	txnData := fmt.Sprintf("Txn: Pay %v from %s\n", 42, c.PublicKey)
	t := datastore.GetEntityMetadata("txn").Instance().(*Transaction)
	t.Value = 1000
	t.ClientID = c.GetKey()
	t.TransactionType = TxnTypeSend
	t.TransactionData = txnData
	t.CreationDate = common.Now()

	_, err = t.Sign(sigScheme)
	if err != nil {
		fmt.Printf("Error signing\n")
	}
	ctx := common.GetRootContext()
	for i := 0; i < b.N; i++ {
		t.PublicKey = c.PublicKey
		t.VerifySignature(ctx)
	}
}

func BenchmarkTransactionRead(b *testing.B) {
	common.SetupRootContext(node.GetNodeContext())
	client.SetupEntity(memorystore.GetStorageProvider())
	SetupEntity(memorystore.GetStorageProvider())

	ctx := memorystore.WithEntityConnection(context.Background(), transactionEntityMetadata)
	defer memorystore.Close(ctx)

	txn := transactionEntityMetadata.Instance().(*Transaction)
	txn.ChainID = config.GetMainChainID()
	txnIDs := make([]datastore.Key, 0, memorystore.BATCH_SIZE)
	getTxnsFunc := func(ctx context.Context, qe datastore.CollectionEntity) bool {
		txnIDs = append(txnIDs, qe.GetKey())
		return len(txnIDs) != memorystore.BATCH_SIZE
	}

	transactionEntityMetadata.GetStore().IterateCollection(ctx, transactionEntityMetadata, txn.GetCollectionName(), getTxnsFunc)
	txns := datastore.AllocateEntities(memorystore.BATCH_SIZE, transactionEntityMetadata)
	for i := 0; i < b.N; i++ {
		transactionEntityMetadata.GetStore().MultiRead(ctx, transactionEntityMetadata, txnIDs, txns)
	}
}

func B1enchmarkTransactionWrite(t *testing.B) {
	common.SetupRootContext(node.GetNodeContext())
	client.SetupEntity(memorystore.GetStorageProvider())
	SetupEntity(memorystore.GetStorageProvider())
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
		sigScheme := encryption.NewED25519Scheme()
		err := sigScheme.GenerateKeys()
		if err != nil {
			panic(err)
		}
		sigSchemes = append(sigSchemes, sigScheme)
		go postClient(sigScheme, done)
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

func postClient(sigScheme encryption.SignatureScheme, done chan<- bool) {
	entity := client.Provider()
	c, ok := entity.(*client.Client)
	if !ok {
		fmt.Printf("it's not ok!\n")
	}
	c.SetPublicKey(sigScheme.GetPublicKey())

	ctx := datastore.WithAsyncChannel(context.Background(), client.ClientEntityChannel)
	//ctx := memorystore.WithEntityConnection(context.Background(),entity.GetEntityMetadata())
	_, err := client.PutClient(ctx, entity)
	//memorystore.Close(ctx)
	if err != nil {
		fmt.Printf("error for %v : %v\n", c.PublicKey, err)
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
	signature, err := t.Sign(c.GetSignatureScheme())
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

func TestExemptedSCFunctions(t *testing.T) {
	txn := &Transaction{}
	invalidFeeMessage := "invalid_request: Invalid request (The given fee is less than the minimum required fee to process the txn)"

	t.Run("min fee is zero and fee is zero", func(t *testing.T) {
		err := txn.ValidateFee()
		require.NoError(t, err)
		require.Zero(t, TXN_MIN_FEE, "min fee is zero")
		require.Zero(t, txn.Fee, "min fee is zero")
	})

	TXN_MIN_FEE = 10

	t.Run("min fee is not zero and fee is zero", func(t *testing.T) {
		err := txn.ValidateFee()
		require.Error(t, err)
		require.EqualError(t, err, invalidFeeMessage)
	})

	t.Run("testing excemptions when true", func(t *testing.T) {
		testExcempts(t, txn, "")
	})

	setExcemptsToFalse()

	t.Run("testing excemptions when false", func(t *testing.T) {
		testExcempts(t, txn, invalidFeeMessage)
	})

	t.Run("test function that isn't exempted", func(t *testing.T) {
		smartContractData := smartContractTransactionData{FunctionName: "unexmpted_sc_function"}
		dataBytes, err := json.Marshal(smartContractData)
		require.NoError(t, err)
		txn.TransactionData = string(dataBytes)
		err = txn.ValidateFee()
		require.Error(t, err)
		require.EqualError(t, err, invalidFeeMessage)
	})

}

func testExcempts(t *testing.T, txn *Transaction, errMessage string) {
	var smartContractData smartContractTransactionData
	for name := range exemptedSCFunctions {
		smartContractData.FunctionName = name
		dataBytes, err := json.Marshal(smartContractData)
		require.NoError(t, err)
		txn.TransactionData = string(dataBytes)
		err = txn.ValidateFee()
		if errMessage == "" {
			require.NoError(t, err)
		} else {
			require.Error(t, err)
			require.EqualError(t, err, errMessage)
		}
	}
}

func setExcemptsToFalse() {
	for name := range exemptedSCFunctions {
		exemptedSCFunctions[name] = false
	}
}
