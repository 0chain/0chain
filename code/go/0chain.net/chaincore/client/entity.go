package client

import (
	"context"
	"encoding/hex"
	"errors"
	"time"

	"0chain.net/chaincore/threshold/bls"
	"0chain.net/core/cache"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/core/memorystore"
)

var clientSignatureScheme string

// SetClientSignatureScheme - set the signature scheme to be used by the client
func SetClientSignatureScheme(scheme string) {
	clientSignatureScheme = scheme
}

var cacher cache.Cache

func init() {
	cacher = cache.NewLFUCache(10 * 1024)
}

// Client - data structure that holds the client data
type Client struct {
	datastore.CollectionMemberField `json:"-" msgpack:"-"`
	datastore.IDField               `yaml:",inline"`
	datastore.VersionField
	datastore.CreationDateField
	PublicKey      string                     `yaml:"public_key" json:"public_key"`
	PublicKeyBytes []byte                     `json:"-" msgpack:"-"`
	SigScheme      encryption.SignatureScheme `json:"-" msgpack:"-"`
}

// NewClient - create a new client object
func NewClient() *Client {
	return datastore.GetEntityMetadata("client").Instance().(*Client)
}

// Clone returns a clone of the Client.
func (c *Client) Clone() *Client {
	if c == nil {
		return nil
	}

	clone := Client{
		IDField:           c.IDField,
		VersionField:      c.VersionField,
		CreationDateField: c.CreationDateField,
		CollectionMemberField: datastore.CollectionMemberField{
			CollectionScore: c.CollectionMemberField.CollectionScore,
		},
	}

	clone.SetPublicKey(c.PublicKey)

	if c.EntityCollection != nil {
		clone.EntityCollection = c.EntityCollection.Clone()
	}

	return &clone
}

var clientEntityMetadata *datastore.EntityMetadataImpl

// GetEntityMetadata - implementing the interface
func (c *Client) GetEntityMetadata() datastore.EntityMetadata {
	return clientEntityMetadata
}

// Validate - implementing the interface
func (c *Client) Validate(ctx context.Context) error {
	if datastore.IsEmpty(c.ID) {
		return common.InvalidRequest("client id is required")
	}
	if !datastore.IsEqual(c.ID, datastore.ToKey(encryption.Hash(c.PublicKeyBytes))) {
		return common.InvalidRequest("client id is not a SHA3-256 hash of the public key")
	}
	return nil
}

// Read - store read
func (c *Client) Read(ctx context.Context, key datastore.Key) error {
	return c.GetEntityMetadata().GetStore().Read(ctx, key, c)
}

// Write - store read
func (c *Client) Write(ctx context.Context) error {
	return c.GetEntityMetadata().GetStore().Write(ctx, c)
}

// Delete - store read
func (c *Client) Delete(ctx context.Context) error {
	return c.GetEntityMetadata().GetStore().Delete(ctx, c)
}

// Verify - given a signature and hash verify it with client's public key
func (c *Client) Verify(signature string, hash string) (bool, error) {
	return c.GetSignatureScheme().Verify(signature, hash)
}

// GetSignatureScheme - return the signature scheme used for this client
func (c *Client) GetSignatureScheme() encryption.SignatureScheme {
	if c.SigScheme != nil {
		return c.SigScheme
	}

	c.SetPublicKey(c.PublicKey)
	return c.SigScheme
}

// Provider - entity provider for client object
func Provider() datastore.Entity {
	c := &Client{}
	c.Version = "1.0"
	c.InitializeCreationDate()
	c.EntityCollection = cliEntityCollection
	return c
}

// ComputeProperties - implement interface
func (c *Client) ComputeProperties() {
	c.EntityCollection = cliEntityCollection
	c.computePublicKeyBytes()
}

func (c *Client) computePublicKeyBytes() {
	b, _ := hex.DecodeString(c.PublicKey)
	c.PublicKeyBytes = b
	c.ID = encryption.Hash(b)
}

// SetPublicKey - set the public key
func (c *Client) SetPublicKey(key string) {
	c.PublicKey = key
	c.computePublicKeyBytes()
	var ss = encryption.GetSignatureScheme(clientSignatureScheme)
	if err := ss.SetPublicKey(c.PublicKey); err != nil {
		panic(err)
	}
	c.SigScheme = ss
}

// GetBLSPublicKey returns the *bls.PublicKey
func (c *Client) GetBLSPublicKey() (*bls.PublicKey, error) {
	if c.SigScheme == nil {
		if c.PublicKey == "" {
			return nil, errors.New("client has no public key")
		}

		// lazy decoding public key
		c.SetPublicKey(c.PublicKey)
	}

	sig, ok := c.SigScheme.(*encryption.BLS0ChainScheme)
	if !ok {
		return nil, errors.New("invalid signature scheme")
	}

	return sig.GetBLSPublicKey(), nil
}

// SetupEntity - setup the entity
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

// GetClients - given a set of client ids, return the clients
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

// GetClient - gets client from either cache or database
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

// PutClient - Given a client data, it stores it
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

// GetIDFromPublicKey computes the ID of a public key
func GetIDFromPublicKey(pubkey string) (string, error) {
	b, err := hex.DecodeString(pubkey)
	if err != nil {
		return "", err
	}

	return encryption.Hash(b), nil
}

var cliEntityCollection *datastore.EntityCollection
