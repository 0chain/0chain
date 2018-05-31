package memorystore

import (
	"context"
	"fmt"
	"testing"
	"time"

	"0chain.net/common"
	"0chain.net/datastore"
)

/*Company - a test data type */
type Company struct {
	CollectionIDField
	//ID     Key    `json:"id"`
	Domain string `json:"domain"`
	Name   string `json:"name,omitempty"`
}

func (c *Company) GetEntityName() string {
	return "company"
}

var companyEntityMetadata = &datastore.EntityMetadataImpl{Name: "company", MemoryDB: "company", Provider: CompanyProvider}

func init() {
	AddPool("company", DefaultPool)
}

/*GetEntityMetadata - implementing the interface */
func (c *Company) GetEntityMetadata() datastore.EntityMetadata {
	return companyEntityMetadata
}

/*
func (c *Company) SetKey(key Key) {
	c.ID = key
}

func (c *Company) GetKey() Key {
	return c.ID
}

func (c *Company) ComputeProperties() {

}

func (c *Company) Validate(ctx context.Context) error {
	return nil
} */

func (c *Company) Read(ctx context.Context, id datastore.Key) error {
	return Read(ctx, id, c)
}

func (c *Company) Write(ctx context.Context) error {
	return Write(ctx, c)
}

func (c *Company) Delete(ctx context.Context) error {
	return Delete(ctx, c)
}

var companyEntityCollection = &EntityCollection{CollectionName: "collection.company", CollectionSize: 10000, CollectionDuration: time.Hour}

/*TransactionProvider - entity provider for client object */
func CompanyProvider() datastore.Entity {
	c := &Company{}
	c.CollectionIDField.EntityCollection = companyEntityCollection
	return c
}

func TestEntityWriteRead(t *testing.T) {
	fmt.Printf("time : %v\n", time.Now().UnixNano()/int64(time.Millisecond))
	common.SetupRootContext(context.Background())
	ctx := WithEntityConnection(common.GetRootContext(), companyEntityMetadata)
	defer Close(ctx)
	zeroChain := CompanyProvider().(*Company)
	zeroChain2 := CompanyProvider().(*Company)
	keys := []datastore.Key{datastore.ToKey([]byte("0chain.net")), datastore.ToKey("0chain.io")}
	entities := []MemoryEntity{zeroChain, zeroChain2}
	fmt.Printf("keys : %v\n", keys)
	err := MultiRead(ctx, companyEntityMetadata, keys, entities)
	if err != nil {
		fmt.Printf("error reading : %v\n", err)
	} else {
		fmt.Printf("e1 %v\n", entities[0])
		fmt.Printf("e2 %v\n", entities[1])
	}
	zeroChain.Domain = "0chain.net"
	zeroChain.Name = "0chain"
	zeroChain.ID = datastore.ToKey(zeroChain.Domain)
	zeroChain.CollectionIDField.EntityCollection = companyEntityCollection
	err = InsertIfNE(ctx, zeroChain)
	if err != nil {
		fmt.Printf("error ifne: %v\n", err)
	}
	zeroChain2.Domain = "0chain.io"
	err = Read(ctx, datastore.ToKey(zeroChain2.Domain), zeroChain2)
	if err != nil {
		fmt.Printf("error reading: %v\n", err)
	} else {
		fmt.Printf("zc2 = %v\n", zeroChain2)
	}
	zeroChain2.InitCollectionScore()
	zeroChain2.SetCollectionScore(zeroChain2.GetCollectionScore() + 10)
	MultiWrite(ctx, companyEntityMetadata, []MemoryEntity{zeroChain, zeroChain2})

	fmt.Printf("iterating\n")
	IterateCollection(ctx, zeroChain.GetCollectionName(), PrintIterator, companyEntityMetadata)
}

/*
func TestEntityCollectionTrimmer(t *testing.T) {
	CollectionTrimmer("collection.company", 100, time.Second)
} */
