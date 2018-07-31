package client

import (
	"context"
	"encoding/hex"
	"time"

	"0chain.net/cache"
	"0chain.net/common"
	"0chain.net/datastore"
	"0chain.net/encryption"
	"0chain.net/memorystore"
)

var cacher cache.Cache

func init() {
	cacher = cache.GetLFUCacheProvider()
	cacher.New(256)
}

/*Client - data structure that holds the client data */
type Client struct {
	datastore.IDField
	datastore.VersionField
	datastore.CreationDateField
	PublicKey      string               `json:"public_key"`
	PublicKeyBytes encryption.HashBytes `json:"-"`
}

var clientEntityMetadata *datastore.EntityMetadataImpl

/*GetEntityMetadata - implementing the interface */
func (c *Client) GetEntityMetadata() datastore.EntityMetadata {
	return clientEntityMetadata
}

/*Validate - implementing the interface */
func (c *Client) Validate(ctx context.Context) error {
	if datastore.IsEmpty(c.ID) {
		return common.InvalidRequest("client id is required")
	}
	if !datastore.IsEqual(c.ID, datastore.ToKey(encryption.Hash(c.PublicKeyBytes))) {
		return common.InvalidRequest("client id is not a SHA3-256 hash of the public key")
	}
	return nil
}

/*GetClient - gets client from either cache or database*/
func (c *Client) GetClient(ctx context.Context, key datastore.Key) error {
	var ok bool
	var co *Client
	ico, cerr := cacher.Get(key)
	if cerr == nil {
		co, ok = ico.(*Client)
	}
	if !ok {
		err := c.Read(ctx, key)
		if err == nil {
			cacher.Add(key, c)
		}
		return err
	} else {
		c.ID = co.ID
		c.PublicKey = co.PublicKey
		c.SetPublicKey(c.PublicKey)
	}
	return nil
}

/*Read - store read */
func (c *Client) Read(ctx context.Context, key datastore.Key) error {
	return c.GetEntityMetadata().GetStore().Read(ctx, key, c)
}

/*Write - store read */
func (c *Client) Write(ctx context.Context) error {
	return c.GetEntityMetadata().GetStore().Write(ctx, c)
}

/*Delete - store read */
func (c *Client) Delete(ctx context.Context) error {
	return c.GetEntityMetadata().GetStore().Delete(ctx, c)
}

/*Verify - given a signature and hash verify it with client's public key */
func (c *Client) Verify(signature string, hash string) (bool, error) {
	return encryption.Verify(c.PublicKeyBytes, signature, hash)
}

/*Provider - entity provider for client object */
func Provider() datastore.Entity {
	c := &Client{}
	c.Version = "1.0"
	c.InitializeCreationDate()
	return c
}

/*ComputeProperties - implement interface */
func (c *Client) ComputeProperties() {
	c.computePublicKeyBytes(c.PublicKey)
}

func (c *Client) computePublicKeyBytes(key string) {
	b, _ := hex.DecodeString(key)
	if len(b) > len(c.PublicKeyBytes) {
		b = b[len(b)-encryption.HASH_LENGTH:]
	}
	copy(c.PublicKeyBytes[encryption.HASH_LENGTH-len(b):], b)
}

/*SetPublicKey - set the public key */
func (c *Client) SetPublicKey(key string) {
	c.PublicKey = key
	c.computePublicKeyBytes(key)
	c.ID = encryption.Hash(c.PublicKeyBytes)
}

/*SetupEntity - setup the entity */
func SetupEntity(store datastore.Store) {
	clientEntityMetadata = datastore.MetadataProvider()
	clientEntityMetadata.Name = "client"
	clientEntityMetadata.Provider = Provider
	clientEntityMetadata.Store = store

	datastore.RegisterEntityMetadata("client", clientEntityMetadata)

	var chunkingOptions = datastore.ChunkingOptions{
		EntityMetadata:   clientEntityMetadata,
		EntityBufferSize: 1024,
		MaxHoldupTime:    500 * time.Millisecond,
		NumChunkCreators: 1,
		ChunkSize:        64,
		ChunkBufferSize:  16,
		NumChunkStorers:  2,
	}
	ClientEntityChannel = memorystore.SetupWorkers(common.GetRootContext(), &chunkingOptions)
}

func SetupEntityForWallet(store datastore.Store) {
	clientEntityMetadata = datastore.MetadataProvider()
	clientEntityMetadata.Name = "client"
	clientEntityMetadata.Provider = Provider
	clientEntityMetadata.Store = store

	datastore.RegisterEntityMetadata("client", clientEntityMetadata)
}

var ClientEntityChannel chan datastore.QueuedEntity

/*GetClients - given a set of client ids, return the clients */
func GetClients(ctx context.Context, clients map[string]*Client) {
	clientIDs := make([]string, len(clients))
	idx := 0
	for key := range clients {
		clientIDs[idx] = key
		idx++
	}
	for i, start := 0, 0; start < len(clients); start += memorystore.BATCH_SIZE {
		end := start + memorystore.BATCH_SIZE
		if end > len(clients) {
			end = len(clients)
		}
		cEntities := make([]datastore.Entity, end-start)
		for j := 0; j < len(cEntities); j++ {
			cEntities[j] = clientEntityMetadata.Instance().(*Client)
		}
		clientEntityMetadata.GetStore().MultiRead(ctx, clientEntityMetadata, clientIDs[start:end], cEntities)
		for j := 0; i < end; i, j = i+1, j+1 {
			clients[clientIDs[i]] = cEntities[j].(*Client)
		}
	}
}
