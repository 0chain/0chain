package client

import (
	"context"
	"fmt"
	"testing"
	"time"

	"0chain.net/common"
	"0chain.net/datastore"
	"0chain.net/encryption"
	"0chain.net/memorystore"
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

func postClient(publicKey string, done chan<- bool) {
	entity := Provider()
	client, ok := entity.(*Client)
	if !ok {
		fmt.Printf("it's not ok!\n")
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
