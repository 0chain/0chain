package util

import (
	"context"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestMPTCachingProxy_InsertValue(t *testing.T) {
	mndb := NewMemoryNodeDB()
	mpt := NewMerklePatriciaTrie(mndb, Sequence(0), nil)
	db := NewLevelNodeDB(NewMemoryNodeDB(), mpt.db, false)
	mpt2 := NewMerklePatriciaTrie(db, Sequence(0), mpt.GetRoot())

	wrapped := NewMPTCachingProxy(context.TODO(), mpt2)

	val := &Txn{"1"}
	err := wrapped.Insert(Path("01"), val)
	cached := wrapped.cache["01"]
	assert.Equal(t, val, cached)
	value := &Txn{}
	raw, err := wrapped.GetNodeValue([]byte("01"), value)
	assert.Nil(t, err)
	value = raw.(*Txn)
	assert.Equal(t, val, value)

	size := db.Size(context.TODO())
	assert.Equal(t, int64(0), size)

	val2 := &Txn{"2"}
	_ = wrapped.Insert(Path("01"), val2)

	size = db.Size(context.TODO())
	assert.Equal(t, int64(0), size)

	wrapped.Flush()

	size = db.Size(context.TODO())
	assert.Equal(t, int64(1), size)

}

func TestMPTCachingProxy_GetNodeValueFromCacheWithoutFlush(t *testing.T) {
	mndb := NewMemoryNodeDB()
	mpt := NewMerklePatriciaTrie(mndb, Sequence(0), nil)
	db := NewLevelNodeDB(NewMemoryNodeDB(), mpt.db, false)
	mpt2 := NewMerklePatriciaTrie(db, Sequence(0), mpt.GetRoot())

	wrapped := NewMPTCachingProxy(context.TODO(), mpt2)

	val := &Txn{"1"}
	err := wrapped.Insert(Path("01"), val)
	cached := wrapped.cache["01"]
	assert.Equal(t, val, cached)

	size := db.Size(context.TODO())
	assert.Equal(t, int64(0), size)

	value := &Txn{}
	raw, err := wrapped.GetNodeValue([]byte("01"), value)
	value = raw.(*Txn)
	assert.Nil(t, err)
	assert.Equal(t, val, value)

}

func TestMPTCachingProxy_GetNodeValueFromCacheWithFlush(t *testing.T) {
	mndb := NewMemoryNodeDB()
	mpt := NewMerklePatriciaTrie(mndb, Sequence(0), nil)
	db := NewLevelNodeDB(NewMemoryNodeDB(), mpt.db, false)
	mpt2 := NewMerklePatriciaTrie(db, Sequence(0), mpt.GetRoot())

	wrapped := NewMPTCachingProxy(context.TODO(), mpt2)

	val := &Txn{"1"}
	err := wrapped.Insert(Path("01"), val)
	cached := wrapped.cache["01"]
	assert.Equal(t, val, cached)

	size := db.Size(context.TODO())
	assert.Equal(t, int64(0), size)

	wrapped.Flush()
	value := &Txn{}
	raw, err := wrapped.GetNodeValue([]byte("01"), value)
	txn := raw.(*Txn)
	assert.Nil(t, err)
	assert.Equal(t, val, txn)
}
func TestMPTCachingProxy_GetNodeValueFromCacheWithFlushUpdate(t *testing.T) {
	mndb := NewMemoryNodeDB()
	mpt := NewMerklePatriciaTrie(mndb, Sequence(0), nil)
	db := NewLevelNodeDB(NewMemoryNodeDB(), mpt.db, false)
	mpt2 := NewMerklePatriciaTrie(db, Sequence(0), mpt.GetRoot())

	wrapped := NewMPTCachingProxy(context.TODO(), mpt2)

	val := &Txn{"1"}
	err := wrapped.Insert(Path("01"), val)
	cached := wrapped.cache["01"]
	assert.Equal(t, val, cached)

	size := db.Size(context.TODO())
	assert.Equal(t, int64(0), size)

	wrapped.Flush()

	val2 := &Txn{"2"}
	err = wrapped.Insert(Path("01"), val2)
	cached2 := wrapped.cache["01"]
	assert.Equal(t, val2, cached2)

	value := &Txn{}
	raw, err := wrapped.GetNodeValue([]byte("01"), value)
	value = raw.(*Txn)
	assert.Nil(t, err)
	assert.Equal(t, val2, value)
}

func TestMPTCachingProxy_GetNodeValueFromCacheNotPresent(t *testing.T) {
	mndb := NewMemoryNodeDB()
	mpt := NewMerklePatriciaTrie(mndb, Sequence(0), nil)
	db := NewLevelNodeDB(NewMemoryNodeDB(), mpt.db, false)
	mpt2 := NewMerklePatriciaTrie(db, Sequence(0), mpt.GetRoot())

	wrapped := NewMPTCachingProxy(context.TODO(), mpt2)

	size := db.Size(context.TODO())
	assert.Equal(t, int64(0), size)
	value := &Txn{}
	_, err := wrapped.GetNodeValue([]byte("01"), value)
	assert.Error(t, err)
}

func TestMPTCachingProxy_DeleteNodeValueFromCache(t *testing.T) {
	mndb := NewMemoryNodeDB()
	mpt := NewMerklePatriciaTrie(mndb, Sequence(0), nil)
	db := NewLevelNodeDB(NewMemoryNodeDB(), mpt.db, false)
	mpt2 := NewMerklePatriciaTrie(db, Sequence(0), mpt.GetRoot())

	wrapped := NewMPTCachingProxy(context.TODO(), mpt2)

	val := &Txn{"1"}
	err := wrapped.Insert(Path("01"), val)
	cached := wrapped.cache["01"]
	assert.Equal(t, val, cached)

	size := db.Size(context.TODO())
	assert.Equal(t, int64(0), size)

	err = wrapped.Delete(Path("01"))
	assert.Nil(t, err)
	value := &Txn{}
	_, err = wrapped.GetNodeValue([]byte("01"), value)
	assert.Error(t, err)
}

func TestMPTCachingProxy_DeleteNodeValueFromCacheWithFlush(t *testing.T) {
	mndb := NewMemoryNodeDB()
	mpt := NewMerklePatriciaTrie(mndb, Sequence(0), nil)
	db := NewLevelNodeDB(NewMemoryNodeDB(), mpt.db, false)
	mpt2 := NewMerklePatriciaTrie(db, Sequence(0), mpt.GetRoot())

	wrapped := NewMPTCachingProxy(context.TODO(), mpt2)

	val := &Txn{"1"}
	err := wrapped.Insert(Path("01"), val)
	cached := wrapped.cache["01"]
	assert.Equal(t, val, cached)

	wrapped.Flush()

	err = wrapped.Delete(Path("01"))
	assert.Nil(t, err)

	size := db.Size(context.TODO())
	assert.Equal(t, int64(0), size)
	value := &Txn{}
	_, err = wrapped.GetNodeValue([]byte("01"), value)
	assert.Error(t, err)

}
func TestMPTCachingProxy_DeleteNodeValueNotPresent(t *testing.T) {
	mndb := NewMemoryNodeDB()
	mpt := NewMerklePatriciaTrie(mndb, Sequence(0), nil)
	db := NewLevelNodeDB(NewMemoryNodeDB(), mpt.db, false)
	mpt2 := NewMerklePatriciaTrie(db, Sequence(0), mpt.GetRoot())

	wrapped := NewMPTCachingProxy(context.TODO(), mpt2)

	err := wrapped.Delete(Path("01"))
	assert.Error(t, err)

}
