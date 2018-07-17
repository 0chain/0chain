package client

import (
	"context"
	"time"

	"0chain.net/common"
	"0chain.net/datastore"
	"0chain.net/encryption"
	"0chain.net/memorystore"
)

/*Client - data structure that holds the client data */
type Client struct {
	datastore.IDField
	datastore.VersionField
	datastore.CreationDateField
	PublicKey      string `json:"public_key"`
	PublicKeyBytes encryption.HashBytes
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
