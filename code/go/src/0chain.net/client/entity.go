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
	datastore.CreationDateField
	PublicKey string `json:"public_key"`
}

var clientEntityMetadata = &datastore.EntityMetadataImpl{Name: "client", MemoryDB: "clientdb", Provider: Provider}

func init() {
	memorystore.AddPool("clientdb", memorystore.DefaultPool)
	//memorystore.AddPool("clientdb", memorystore.NewPool(":6380"))
}

/*GetEntityMetadata - implementing the interface */
func (c *Client) GetEntityMetadata() datastore.EntityMetadata {
	return clientEntityMetadata
}

/*GetEntityName - implementing the interface */
func (c *Client) GetEntityName() string {
	return "client"
}

/*Validate - implementing the interface */
func (c *Client) Validate(ctx context.Context) error {
	if datastore.IsEmpty(c.ID) {
		return common.InvalidRequest("client id is required")
	}
	if !datastore.IsEqual(c.ID, datastore.ToKey(encryption.Hash(c.PublicKey))) {
		return common.InvalidRequest("client id is not a SHA3-256 hash of the public key")
	}
	return nil
}

/*Read - store read */
func (c *Client) Read(ctx context.Context, key datastore.Key) error {
	return memorystore.Read(ctx, key, c)
}

/*Write - store read */
func (c *Client) Write(ctx context.Context) error {
	return memorystore.Write(ctx, c)
}

/*Delete - store read */
func (c *Client) Delete(ctx context.Context) error {
	return memorystore.Delete(ctx, c)
}

/*Verify - given a signature and hash verify it with client's public key */
func (c *Client) Verify(signature string, hash string) (bool, error) {
	return encryption.Verify(c.PublicKey, signature, hash)
}

/*Provider - entity provider for client object */
func Provider() datastore.Entity {
	c := &Client{}
	c.InitializeCreationDate()
	return c
}

/*SetupEntity - setup the entity */
func SetupEntity() {
	datastore.RegisterEntityMetadata("client", clientEntityMetadata)

	var chunkingOptions = memorystore.ChunkingOptions{
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

var ClientEntityChannel chan memorystore.MemoryEntity

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
		cEntities := make([]memorystore.MemoryEntity, end-start+1)
		for j := 0; j < len(cEntities); j++ {
			cEntities[j] = clientEntityMetadata.Instance().(*Client)
		}
		memorystore.MultiRead(ctx, clientEntityMetadata, clientIDs[start:end], cEntities)
		for j := 0; i < end; i, j = i+1, j+1 {
			clients[clientIDs[i]] = cEntities[j].(*Client)
		}
	}
}
