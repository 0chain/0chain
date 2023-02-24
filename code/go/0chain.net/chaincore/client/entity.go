package client

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"0chain.net/core/cache"
	"github.com/0chain/common/core/logging"
	"go.uber.org/zap"

	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/core/memorystore"
	"github.com/herumi/bls/ffi/go/bls"
)

var defaultClientSignatureScheme = encryption.SignatureSchemeBls0chain

// SetClientSignatureScheme - set the signature scheme to be used by the client
func SetClientSignatureScheme(scheme string) {
	defaultClientSignatureScheme = scheme
}

var cacher cache.Cache

func init() {
	cacher = cache.NewLFUCache(10 * 1024)
}

// SetupClientDB sets up client DB
func SetupClientDB() {
	memorystore.AddPool("clientdb", memorystore.DefaultPool)
}

// Client - data structure that holds the client data
//
//go:generate msgp -io=false -tests=false -v
type Client struct {
	datastore.CollectionMemberField `json:"-" msgpack:"-" msg:"-" yaml:"-"`
	datastore.IDField               `yaml:",inline"`
	datastore.VersionField          `yaml:"-"`
	datastore.CreationDateField     `yaml:"-"`
	PublicKey                       string                     `yaml:"public_key" json:"public_key"`
	PublicKeyBytes                  []byte                     `json:"-" msgpack:"-" msg:"-" yaml:"-"`
	sigSchemeType                   string                     `yaml:"-"`
	SigScheme                       encryption.SignatureScheme `json:"-" msgpack:"-" msg:"-" yaml:"-"`
}

// NewClient - create a new client object
func NewClient(opts ...Option) *Client {
	cli := datastore.GetEntityMetadata("client").Instance().(*Client)
	for _, opt := range opts {
		opt(cli)
	}

	if cli.sigSchemeType == "" {
		cli.sigSchemeType = defaultClientSignatureScheme
	}

	return cli
}

// Clone returns a clone of the Client.
func (c *Client) Clone() *Client {
	if c == nil {
		return nil
	}

	clone := &Client{}
	clone.Copy(c)
	return clone
}

func (c *Client) Copy(src *Client) {
	c.IDField = src.IDField
	c.VersionField = src.VersionField
	c.CreationDateField = src.CreationDateField
	c.sigSchemeType = src.sigSchemeType
	c.CollectionMemberField = datastore.CollectionMemberField{
		CollectionScore: src.CollectionMemberField.CollectionScore,
	}

	if err := c.SetPublicKey(src.PublicKey); err != nil {
		logging.Logger.Error("client copy failed on setting public key", zap.Error(err))
	}

	if src.EntityCollection != nil {
		c.EntityCollection = src.EntityCollection.Clone()
	}
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
	ss, err := c.GetSignatureScheme()
	if err != nil {
		return false, err
	}

	return ss.Verify(signature, hash)
}

// GetSignatureScheme - return the signature scheme used for this client
func (c *Client) GetSignatureScheme() (encryption.SignatureScheme, error) {
	if c.SigScheme != nil {
		return c.SigScheme, nil
	}

	if err := c.SetPublicKey(c.PublicKey); err != nil {
		return nil, fmt.Errorf("client got invalid public key, err: %v", err)
	}
	return c.SigScheme, nil
}

// Provider - entity provider for client object
func Provider() datastore.Entity {
	c := &Client{}
	c.Version = "1.0"
	c.sigSchemeType = defaultClientSignatureScheme
	c.InitializeCreationDate()
	c.EntityCollection = cliEntityCollection
	return c
}

// ComputeProperties - implement interface
func (c *Client) ComputeProperties() error {
	c.EntityCollection = cliEntityCollection
	return c.computePublicKeyBytes()
}

func (c *Client) computePublicKeyBytes() error {
	b, err := hex.DecodeString(c.PublicKey)
	if err != nil {
		return err
	}
	c.PublicKeyBytes = b
	c.ID = encryption.Hash(b)
	return nil
}

// SetPublicKey - set the public key
func (c *Client) SetPublicKey(key string) error {
	oldPK := c.PublicKey
	c.PublicKey = key
	if err := c.computePublicKeyBytes(); err != nil {
		c.PublicKey = oldPK
		return err
	}

	sigSchemeType := c.sigSchemeType
	if sigSchemeType == "" {
		sigSchemeType = defaultClientSignatureScheme
	}

	var ss = encryption.GetSignatureScheme(sigSchemeType)
	if err := ss.SetPublicKey(c.PublicKey); err != nil {
		return err
	}
	c.SigScheme = ss
	return nil
}

// SetSignatureScheme sets the signature scheme
func (c *Client) SetSignatureScheme(sig encryption.SignatureScheme) error {
	c.PublicKey = sig.GetPublicKey()
	if err := c.computePublicKeyBytes(); err != nil {
		return err
	}
	c.SigScheme = sig
	switch sig.(type) {
	case *encryption.ED25519Scheme:
		c.sigSchemeType = encryption.SignatureSchemeEd25519
	case *encryption.BLS0ChainScheme:
		c.sigSchemeType = encryption.SignatureSchemeBls0chain
	default:
		return encryption.ErrInvalidSignatureScheme
	}

	return nil
}

// SetSignatureSchemeType sets the signature scheme type
func (c *Client) SetSignatureSchemeType(v string) {
	c.sigSchemeType = v
}

// GetBLSPublicKey returns the *bls.PublicKey
func (c *Client) GetBLSPublicKey() (*bls.PublicKey, error) {
	if c.SigScheme == nil {
		if c.PublicKey == "" {
			return nil, errors.New("client has no public key")
		}

		// lazy decoding public key
		if err := c.SetPublicKey(c.PublicKey); err != nil {
			return nil, err
		}
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
	clientEntityMetadata.DB = "clientdb"
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

// GetClientFromCache - gets client from either cache
func GetClientFromCache(key datastore.Key) (*Client, error) {
	co, err := cacher.Get(key)
	if err != nil {
		return nil, err
	}
	return co.(*Client), nil
}

// PutClientCache saves client to cache
func PutClientCache(co *Client) error {
	return cacher.Add(co.GetKey(), co)
}

// GetClient - gets client from either cache or database
func GetClient(ctx context.Context, key datastore.Key) (*Client, error) {
	coi, err := cacher.Get(key)
	if err == nil {
		return coi.(*Client), nil
	}

	co := NewClient()
	if err = co.Read(ctx, key); err != nil {
		return nil, err
	}

	if err := cacher.Add(key, co); err != nil {
		return nil, err
	}

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
	if err := cacher.Add(co.GetKey(), co); err != nil {
		logging.Logger.Warn("put client to cache failed", zap.Error(err))
	}
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

// Option represents the optional parameters type for creating
// a client instance
type Option func(opt *Client)

// SignatureScheme is the option for setting client's signature scheme name
func SignatureScheme(schemeType string) Option {
	return func(opt *Client) {
		opt.sigSchemeType = schemeType
	}
}
