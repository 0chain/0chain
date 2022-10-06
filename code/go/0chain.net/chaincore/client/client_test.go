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
	"github.com/0chain/common/core/logging"
	"github.com/alicebob/miniredis/v2"
	"github.com/gomodule/redigo/redis"
	"github.com/stretchr/testify/require"
	"github.com/vmihailenco/msgpack/v5"
)

func init() {
	logging.InitLogging("testing", "")
}

func initDefaultPool() error {
	mr, err := miniredis.Run()
	if err != nil {
		return err
	}

	memorystore.DefaultPool = &redis.Pool{
		MaxIdle:   80,
		MaxActive: 1000, // max number of connections
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", mr.Addr())
			if err != nil {
				panic(err.Error())
			}
			return c, err
		},
	}

	return nil
}

func setupEntity() {
	em := datastore.EntityMetadataImpl{
		Name:     "client",
		DB:       "clientdb",
		Store:    memorystore.GetStorageProvider(),
		Provider: Provider,
	}
	clientEntityMetadata = &em
	datastore.RegisterEntityMetadata("client", &em)

	memorystore.AddPool(em.DB, memorystore.DefaultPool)

	cliEntityCollection = &datastore.EntityCollection{CollectionName: "collection.cli", CollectionSize: 60000000000, CollectionDuration: time.Minute}
}

func TestSaveClients(t *testing.T) {
	common.SetupRootContext(context.Background())
	if err := initDefaultPool(); err != nil {
		t.Fatal(err)
	}
	setupEntity()

	publicKey := "627eb53becc3d312836bfdd97deb25a6d71f1e15bf3bcd233ab3d0c36300161990d4e2249f1d7747c0d1775ee7ffec912a61bd8ab5ed164fd6218099419c4305"
	client := NewClient(SignatureScheme(encryption.SignatureSchemeEd25519))
	require.NoError(t, client.SetPublicKey(publicKey))

	v, err := msgpack.Marshal(client)
	require.NoError(t, err)

	var c Client
	err = msgpack.Unmarshal(v, &c)
	require.NoError(t, err)

	require.Equal(t, client.PublicKey, c.PublicKey)
}

func TestClientChunkSave(t *testing.T) {
	common.SetupRootContext(context.Background())
	if err := initDefaultPool(); err != nil {
		t.Fatal(err)
	}
	setupEntity()
	numWorkers := 1000
	done := make(chan bool, 100)
	for i := 1; i <= numWorkers; i++ {
		sigScheme := encryption.NewED25519Scheme()
		err := sigScheme.GenerateKeys()
		if err != nil {
			t.Fatal(err)
		}
		go postClient(t, sigScheme, done)
	}
	for count := 0; true; {
		<-done
		count++
		if count == numWorkers {
			break
		}
	}
	common.Done()
}

func TestClientID(t *testing.T) {
	setupEntity()
	publicKey := "627eb53becc3d312836bfdd97deb25a6d71f1e15bf3bcd233ab3d0c36300161990d4e2249f1d7747c0d1775ee7ffec912a61bd8ab5ed164fd6218099419c4305"
	client := NewClient(SignatureScheme(encryption.SignatureSchemeEd25519))
	require.NoError(t, client.SetPublicKey(publicKey))
}

func postClient(t *testing.T, sigScheme encryption.SignatureScheme, done chan<- bool) {
	var client *Client
	switch sigScheme.(type) {
	case *encryption.ED25519Scheme:
		client = NewClient(SignatureScheme(encryption.SignatureSchemeEd25519))
	case *encryption.BLS0ChainScheme:
		client = NewClient(SignatureScheme(encryption.SignatureSchemeBls0chain))
	}

	pk := sigScheme.GetPublicKey()
	require.NoError(t, client.SetPublicKey(pk))
	ctx := datastore.WithAsyncChannel(context.Background(), ClientEntityChannel)
	ctx = memorystore.WithConnection(ctx)
	_, err := PutClient(ctx, client)
	if err != nil {
		fmt.Printf("error for %v : %v %v\n", pk, client.GetKey(), err)
	}
	done <- true
}
