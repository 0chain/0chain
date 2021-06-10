package store

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"time"

	"golang.org/x/sys/unix"
)

const (
	fsStoreIsWritableCheckInterval = time.Second
)

var _ Store = (*fsStore)(nil)

func NewFSStore(dir string) (Store, error) {
	store := &fsStore{dir: dir}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	store.size = store.Size()
	store.count = store.Count()
	return store, nil
}

type fsStore struct {
	dir                 string
	count, size         int64
	isWritable          bool
	isWritableCheckedAt time.Time
}

func (store *fsStore) getPath(key []byte) string {
	key = bytes.TrimSpace(key)
	if len(key) == 0 {
		return ""
	}
	return path.Join(store.dir, string(key))
}

func (store *fsStore) Get(key []byte) (value []byte, err error) {
	value, err = ioutil.ReadFile(store.getPath(key))
	if err != nil {
		if os.IsNotExist(err) {
			err = ErrKeyNotFound
		}
		return nil, err
	}
	return value, nil
}

func (store *fsStore) GetReader(key []byte) (r io.ReadCloser, err error) {
	r, err = os.Open(store.getPath(key))
	if err != nil {
		if os.IsNotExist(err) {
			err = ErrKeyNotFound
		}
		return nil, err
	}
	return r, nil
}

func (store *fsStore) Put(key, value []byte) (err error) {
	var isNew bool
	path := store.getPath(key)
	stat, err := os.Stat(path)
	if os.IsNotExist(err) {
		isNew = true
	}
	err = ioutil.WriteFile(path, value, 0644)
	if err != nil {
		return err
	}
	if isNew {
		store.count++
	}
	if stat != nil {
		store.size -= norm(stat.Size())
	}
	store.size += norm(int64(len(value)))
	return nil
}

func (store *fsStore) PutReader(key []byte, r io.Reader) (err error) {
	var isNew bool
	path := store.getPath(key)
	stat, err := os.Stat(path)
	if os.IsNotExist(err) {
		isNew = true
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	n, err := io.Copy(f, r)
	if err != nil {
		return err
	}
	if isNew {
		store.count++
	}
	if stat != nil {
		store.size -= norm(stat.Size())
	}
	store.size += norm(n)
	return nil
}

func (store *fsStore) Delete(key []byte) (err error) {
	path := store.getPath(key)
	stat, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			err = ErrKeyNotFound
		}
		return err
	}
	if err = os.Remove(path); err != nil {
		return err
	}
	store.count--
	store.size -= norm(stat.Size())
	return nil
}

func (store *fsStore) Count() int64 {
	if store.count == 0 {
		entries, err := ioutil.ReadDir(store.dir)
		if err == nil {
			store.count = int64(len(entries))
		}
	}
	return store.count
}

func (store *fsStore) Size() int64 {
	if store.size == 0 {
		size, err := getDirSize(store.dir)
		if err == nil {
			store.size = size
		}
	}
	return store.size
}

func (store *fsStore) IsExist(key []byte) bool {
	_, err := os.Stat(store.getPath(key))
	return !os.IsNotExist(err)
}

func (store *fsStore) IsWritable() bool {
	now := time.Now()
	if now.Sub(store.isWritableCheckedAt) > fsStoreIsWritableCheckInterval {
		store.isWritable = unix.Access(store.dir, unix.W_OK) == nil
		store.isWritableCheckedAt = now
	}
	return store.isWritable
}

func getDirSize(root string) (size int64, err error) {
	err = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if path == root {
			size += norm(0)
			return nil
		}
		if !info.IsDir() {
			size += norm(info.Size())
			return nil
		}
		dirSize, err := getDirSize(path)
		if err != nil {
			return err
		}
		size += dirSize
		return nil
	})
	return
}

func norm(size int64) int64 {
	blockSize := int64(4 * 1024)
	return blockSize * (1 + size/blockSize)
}
