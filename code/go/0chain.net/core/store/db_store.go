package store

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"go.etcd.io/bbolt"
)

var _ Store = (*dbStore)(nil)

var (
	bucket = []byte("default")
)

func NewDBStore(path string) (Store, error) {
	store := &dbStore{}
	err := os.MkdirAll(filepath.Dir(path), 0755)
	if err != nil {
		return nil, err
	}
	store.db, err = bbolt.Open(path, 0644, nil)
	if err != nil {
		return nil, err
	}
	if err = store.db.Update(func(t *bbolt.Tx) error {
		_, err := t.CreateBucket(bucket)
		return err
	}); err != nil {
		return nil, err
	}
	store.size = store.Size()
	store.count = store.Count()
	return store, nil
}

type dbStore struct {
	db          *bbolt.DB
	size, count int64
}

func (store *dbStore) Get(key []byte) (value []byte, err error) {
	err = store.db.View(func(t *bbolt.Tx) error {
		v := t.Bucket(bucket).Get(key)
		if v == nil {
			return ErrKeyNotFound
		}
		value = make([]byte, len(v))
		copy(value, v)
		return nil
	})
	return
}

func (store *dbStore) GetReader(key []byte) (r io.ReadCloser, err error) {
	err = store.db.View(func(t *bbolt.Tx) error {
		v := t.Bucket(bucket).Get(key)
		if v == nil {
			return ErrKeyNotFound
		}
		value := make([]byte, len(v))
		copy(value, v)
		r = ioutil.NopCloser(bytes.NewBuffer(value))
		return nil
	})
	return
}

func (store *dbStore) Put(key, value []byte) (err error) {
	return store.db.Update(func(t *bbolt.Tx) error {
		v := t.Bucket(bucket).Get(key)
		isNew := v == nil
		err := t.Bucket(bucket).Put(key, value)
		if err != nil {
			return err
		}
		if isNew {
			store.count++
		}
		store.size += int64(len(value) - len(v))
		return nil
	})
}

func (store *dbStore) PutReader(key []byte, r io.Reader) (err error) {
	value, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}
	return store.Put(key, value)
}

func (store *dbStore) Delete(key []byte) (err error) {
	return store.db.Update(func(t *bbolt.Tx) error {
		v := t.Bucket(bucket).Get(key)
		if v == nil {
			return ErrKeyNotFound
		}
		err = t.Bucket(bucket).Delete(key)
		if err != nil {
			return err
		}
		store.count--
		store.size -= int64(len(v))
		return nil
	})
}

func (store *dbStore) Count() (count int64) {
	if store.count == 0 {
		store.db.View(func(t *bbolt.Tx) error {
			store.count = int64(t.Bucket(bucket).Stats().KeyN)
			return nil
		})
	}
	return store.count
}

func (store *dbStore) Size() (size int64) {
	if store.size == 0 {
		store.db.View(func(t *bbolt.Tx) error {
			// update store.size
			return nil
		})
	}
	return store.size
	// return store.db.Stats().TxStats.PageAlloc
}

func (store *dbStore) IsExist(key []byte) bool {
	isExist := false
	store.db.View(func(t *bbolt.Tx) error {
		isExist = t.Bucket(bucket).Get(key) != nil
		return nil
	})
	return isExist
}

func (store *dbStore) IsWritable() bool {
	return !store.db.IsReadOnly()
}
