package client

import (
	"context"
	"fmt"
	"testing"
	"time"

	"0chain.net/datastore"
	"0chain.net/encryption"
)

func BenchmarkClientChunkSave(t *testing.B) {
	fmt.Printf("time : %v\n", time.Now().UnixNano()/int64(time.Millisecond))
	start := time.Now()
	fmt.Printf("Testing at %v\n", start)
	numWorkers := 1
	done := make(chan bool, numWorkers)
	for i := 1; i <= numWorkers; i++ {
		publicKey, privateKey := encryption.GenerateKeys()
		if privateKey == "" {
			fmt.Println("Error genreating keys")
			continue
		}
		go postClient(publicKey, done)
	}
	for count := 0; true; {
		<-done
		count++
		if count == numWorkers {
			break
		}
	}
	fmt.Printf("Elapsed time: %v\n", time.Since(start))
	time.Sleep(time.Second)
}

func postClient(publicKey string, done chan<- bool) {
	entity := ClientProvider()
	client, ok := entity.(*Client)
	if !ok {
		fmt.Printf("it's not ok!\n")
	}
	client.PublicKey = publicKey
	client.SetKey(encryption.Hash(client.PublicKey))

	//ctx := datastore.WithAsyncChannel(context.Background(), ClientEntityChannel)
	ctx := datastore.WithConnection(context.Background())
	_, err := PutClient(ctx, entity)
	if err != nil {
		fmt.Printf("error for %v : %v\n", publicKey, err)
	}
	done <- true
}
