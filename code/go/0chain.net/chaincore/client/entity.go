package client

import (
	"context"
	"encoding/hex"
	"time"

	"github.com/0chain/0chain/code/go/0chain.net/core/cache"
	"github.com/0chain/0chain/code/go/0chain.net/core/common"
	"github.com/0chain/0chain/code/go/0chain.net/core/datastore"
	"github.com/0chain/0chain/code/go/0chain.net/core/encryption"
	"github.com/0chain/0chain/code/go/0chain.net/core/memorystore"
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
	datastore.CollectionMemberField
	datastore.IDField `yaml:",inline"`
	datastore.VersionField
	datastore.CreationDateField
	PublicKey      string `yaml:"public_key" json:"public_key"`
	PublicKeyBytes []byte `json:"-"`
}

//NewClient - create a new client object
func NewClient() *Client {
	return datastore.GetEntityMetadata("client").Instance().(*Client)
}

// Copy of the Client.
func (c *Client) Copy() (cp *Client) {
	cp = new(Client)
	cp.CollectionMemberField.CollectionScore = c.CollectionMemberField.CollectionScore
	cp.CollectionMemberField.EntityCollection = c.CollectionMemberField.EntityCollection.Copy()
	cp.IDField.ID = c.IDField.ID
	cp.VersionField.Version = c.VersionField.Version
	cp.CreationDateField.CreationDate = c.CreationDateField.CreationDate
	cp.PublicKey = c.PublicKey
	if len(c.PublicKeyBytes) > 0 {
		cp.PublicKeyBytes = make([]byte, len(c.PublicKeyBytes))
		copy(cp.PublicKeyBytes, c.PublicKeyBytes)
	}
	return
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
	c.EntityCollection = cliEntityCollection
	return c
}

/*ComputeProperties - implement interface */
func (c *Client) ComputeProperties() {
	c.EntityCollection = cliEntityCollection
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
	cliEntityCollection = &datastore.EntityCollection{CollectionName: "collection.cli", CollectionSize: 60000000000, CollectionDuration: time.Minute}

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
func GetClients(ctx context.Context, clients map[string]*Client) (err error) {

	var (
		clientIDs = make([]string, 0, len(clients))
		cEntities = make([]datastore.Entity, 0, len(clients))
	)

	for key := range clients {
		clientIDs = append(clientIDs, key)
		cEntities = append(cEntities, clientEntityMetadata.Instance().(*Client))
	}

	err = clientEntityMetadata.GetStore().MultiRead(ctx, clientEntityMetadata,
		clientIDs, cEntities)
	if err != nil {
		return
	}

	for _, cl := range cEntities {
		if cl == nil {
			continue
		}
		clients[cl.GetKey()] = cl.(*Client)
	}

	return
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
		return nil, common.NewError("entity_invalid_type", "Invalid entity type")
	}
	response, err := datastore.PutEntityHandler(ctx, entity)
	if err != nil {
		return nil, err
	}
	cacher.Add(co.GetKey(), co)
	return response, nil
}

var cliEntityCollection *datastore.EntityCollection
