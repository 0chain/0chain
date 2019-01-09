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

var clientSignatureScheme string

//SetClientSignatureScheme - set the signature scheme to be used by the client
func SetClientSignatureScheme(scheme string) {
	clientSignatureScheme = scheme
}

var cacher cache.Cache

func init() {
	cacher = cache.NewLFUCache(10 * 1024)
}

/*Client - data structure that holds the client data */
type Client struct {
	datastore.IDField
	datastore.VersionField
	datastore.CreationDateField
	PublicKey      string `json:"public_key"`
	PublicKeyBytes []byte `json:"-"`
}

//NewClient - create a new client object
func NewClient() *Client {
	return datastore.GetEntityMetadata("client").Instance().(*Client)
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
	return c.GetSignatureScheme().Verify(signature, hash)
}

/*GetSignatureScheme - return the signature scheme used for this client */
func (c *Client) GetSignatureScheme() encryption.SignatureScheme {
	var ss = encryption.GetSignatureScheme(clientSignatureScheme)
	ss.SetPublicKey(c.PublicKey)
	return ss
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
	c.computePublicKeyBytes()
}

func (c *Client) computePublicKeyBytes() {
	b, _ := hex.DecodeString(c.PublicKey)
	c.PublicKeyBytes = b
}

/*SetPublicKey - set the public key */
func (c *Client) SetPublicKey(key string) {
	c.PublicKey = key
	c.computePublicKeyBytes()
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

/*GetClient - gets client from either cache or database*/
func GetClient(ctx context.Context, key datastore.Key) (*Client, error) {
	if co, cerr := cacher.Get(key); cerr == nil {
		return co.(*Client), nil
	}
	co := NewClient()
	err := co.Read(ctx, key)
	if err != nil {
		return nil, err
	}
	cacher.Add(key, co)
	return co, nil
}

/*PutClient - Given a client data, it stores it */
func PutClient(ctx context.Context, entity datastore.Entity) (interface{}, error) {
	co, ok := entity.(*Client)
	if !ok {
		return nil, common.NewError("entity_invalid_type", "Invald entity type")
	}
	response, err := datastore.PutEntityHandler(ctx, entity)
	if err != nil {
		return nil, err
	}
	cacher.Add(co.GetKey(), co)
	return response, nil
}
