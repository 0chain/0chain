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

func clearTestFSStore() {
	err := os.RemoveAll(path.Join("tmp", "fs"))
	if err != nil {
		panic(err)
	}
}

func createTestFSStore() Store {
	clearTestFSStore()
	store, err := NewFSStore("tmp/fs")
	if err != nil {
		panic(err)
	}
	return store
}

func TestNewFSStore(t *testing.T) {
	store := createTestFSStore()
	fmt.Println(store.Size())
}

func TestFSStoreGet(t *testing.T) {
	store := createTestFSStore()

	k, v := []byte("hello"), []byte("world")

	err := store.Put(k, v)
	require.NoError(t, err)

	value, err := store.Get(k)
	require.NoError(t, err)
	require.EqualValues(t, v, value)
}

func TestFSStoreGetReader(t *testing.T) {
	store := createTestFSStore()

	k, v := []byte("hello"), []byte("world")

	err := store.Put(k, v)
	require.NoError(t, err)

	r, err := store.GetReader(k)
	require.NoError(t, err)

	value, err := ioutil.ReadAll(r)
	require.NoError(t, err)

	require.EqualValues(t, v, value)
}

func TestFSStorePut(t *testing.T) {
	store := createTestFSStore()

	k, v := []byte("hello"), []byte("world")

	err := store.Put(k, v)
	require.NoError(t, err)

	value, err := store.Get(k)
	require.NoError(t, err)
	require.EqualValues(t, v, value)
}

func TestFSStorePutReader(t *testing.T) {
	store := createTestFSStore()

	k, v := []byte("hello"), []byte("world")

	err := store.PutReader(k, bytes.NewReader(v))
	require.NoError(t, err)

	value, err := store.Get(k)
	require.NoError(t, err)
	require.EqualValues(t, v, value)
}

func TestFSStoreDelete(t *testing.T) {
	store := createTestFSStore()

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

}
