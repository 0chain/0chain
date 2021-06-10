package store

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/require"
)

func clearTestDBStore() {
	err := os.RemoveAll(path.Join("tmp", "db"))
	if err != nil {
		panic(err)
	}
}

func createTestDBStore() Store {
	store, err := NewDBStore("tmp/db/bolt.db")
	if err != nil {
		panic(err)
	}
	return store
}

func TestNewDBStore(t *testing.T) {
	clearTestDBStore()

	store := createTestDBStore()
	fmt.Println(store.Size())
}

func TestDBStoreGet(t *testing.T) {
	clearTestDBStore()

	store := createTestDBStore()

	k, v := []byte("hello"), []byte("world")

	err := store.Put(k, v)
	require.NoError(t, err)

	value, err := store.Get(k)
	require.NoError(t, err)
	require.EqualValues(t, v, value)
}

func TestDBStoreGetReader(t *testing.T) {
	clearTestDBStore()

	store := createTestDBStore()

	k, v := []byte("hello"), []byte("world")

	err := store.Put(k, v)
	require.NoError(t, err)

	r, err := store.GetReader(k)
	require.NoError(t, err)

	value, err := ioutil.ReadAll(r)
	require.NoError(t, err)

	require.EqualValues(t, v, value)
}

func TestDBStorePut(t *testing.T) {
	clearTestDBStore()

	store := createTestDBStore()

	k, v := []byte("hello"), []byte("world")

	err := store.Put(k, v)
	require.NoError(t, err)

	value, err := store.Get(k)
	require.NoError(t, err)
	require.EqualValues(t, v, value)
}

func TestDBStorePutReader(t *testing.T) {
	clearTestDBStore()

	store := createTestDBStore()

	k, v := []byte("hello"), []byte("world")

	err := store.PutReader(k, bytes.NewReader(v))
	require.NoError(t, err)

	value, err := store.Get(k)
	require.NoError(t, err)
	require.EqualValues(t, v, value)
}

func TestDBStoreDelete(t *testing.T) {
	clearTestDBStore()

	store := createTestDBStore()

	k, v := []byte("hello"), []byte("world")

	err := store.Put(k, v)
	require.NoError(t, err)

	value, err := store.Get(k)
	require.NoError(t, err)
	require.EqualValues(t, v, value)

	err = store.Delete(k)
	require.NoError(t, err)

	_, err = store.Get(k)
	require.EqualError(t, err, ErrKeyNotFound.Error())

	fmt.Println(store.Count(), store.Size())
}
