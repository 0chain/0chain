package client

import (
	"context"
	"fmt"
	"testing"
	"time"

	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/core/memorystore"
)

func TestClientChunkSave(t *testing.T) {
	common.SetupRootContext(context.Background())
	SetupEntity(memorystore.GetStorageProvider())
	fmt.Printf("time : %v\n", time.Now().UnixNano()/int64(time.Millisecond))
	start := time.Now()
	fmt.Printf("Testing at %v\n", start)
	numWorkers := 1000
	done := make(chan bool, 100)
	for i := 1; i <= numWorkers; i++ {
		sigScheme := encryption.NewED25519Scheme()
		err := sigScheme.GenerateKeys()
		if err != nil {
			fmt.Printf("Error genreating keys %v\n", err)
			continue
		}
		go postClient(sigScheme.GetPublicKey(), done)
	}
	for count := 0; true; {
		<-done
		count++
		if count == numWorkers {
			break
		}
	}
	time.Sleep(1000 * time.Millisecond)
	common.Done()
	fmt.Printf("Elapsed time: %v\n", time.Since(start))
}

func TestClientID(t *testing.T) {
	publicKey := "627eb53becc3d312836bfdd97deb25a6d71f1e15bf3bcd233ab3d0c36300161990d4e2249f1d7747c0d1775ee7ffec912a61bd8ab5ed164fd6218099419c4305"
	entity := Provider()
	client, ok := entity.(*Client)
	if !ok {
		panic("it's not ok!\n")
	}
	client.SetPublicKey(publicKey)
	fmt.Printf("client id: %v\n", client.ID)
}

func postClient(publicKey string, done chan<- bool) {
	entity := Provider()
	client, ok := entity.(*Client)
	if !ok {
		panic("it's not ok!\n")
	}
	client.SetPublicKey(publicKey)

	ctx := datastore.WithAsyncChannel(context.Background(), ClientEntityChannel)
	//ctx := memorystore.WithEntityConnection(context.Background(), clientEntityMetadata)
	//defer memorystore.Close(ctx)
	_, err := PutClient(ctx, entity)
	if err != nil {
		fmt.Printf("error for %v : %v %v\n", publicKey, client.GetKey(), err)
	}
	done <- true
}
