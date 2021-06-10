package store

import (
	"errors"
	"io"
	"math/rand"
	"sync/atomic"
	"time"
)

type MultiStorePutStrategy byte

const (
	Random MultiStorePutStrategy = iota
	RoundRobin
	MinSizeFirst
	MinCountFirst
)

var (
	ErrStoreUnavailable = errors.New("store unavailable")
)

var _ Store = (*multiStore)(nil)

func NewMultiStore(stores []Store, strategy MultiStorePutStrategy) Store {
	return &multiStore{stores: stores, strategy: strategy}
}

type multiStore struct {
	strategy        MultiStorePutStrategy
	stores          []Store
	roundRobinIndex int64
}

func (store *multiStore) Get(key []byte) (value []byte, err error) {
	for _, s := range store.stores {
		value, err = s.Get(key)
		if err == nil {
			return value, nil
		}
	}
	return nil, err
}

func (store *multiStore) GetReader(key []byte) (r io.ReadCloser, err error) {
	for _, s := range store.stores {
		r, err = s.GetReader(key)
		if err == nil {
			return r, nil
		}
	}
	return nil, err
}

func (store *multiStore) Put(key, value []byte) (err error) {
	s, err := store.pick(store.stores, store.strategy)
	if err != nil {
		return err
	}
	return s.Put(key, value)
}

func (store *multiStore) PutReader(key []byte, r io.Reader) (err error) {
	s, err := store.pick(store.stores, store.strategy)
	if err != nil {
		return err
	}
	return s.PutReader(key, r)
}

func (store *multiStore) Delete(key []byte) (err error) {
	deleted := false
	for _, s := range store.stores {
		if _err := s.Delete(key); _err != nil {
			err = _err
			continue
		}
		deleted = true
	}
	if !deleted {
		return err
	}
	return nil
}

func (store *multiStore) Count() (count int64) {
	for _, s := range store.stores {
		count += s.Count()
	}
	return
}

func (store *multiStore) Size() (size int64) {
	for _, s := range store.stores {
		size += s.Size()
	}
	return
}

func (store *multiStore) IsExist(key []byte) bool {
	for _, s := range store.stores {
		if s.IsExist(key) {
			return true
		}
	}
	return false
}

func (store *multiStore) IsWritable() bool {
	for _, s := range store.stores {
		if s.IsWritable() {
			return true
		}
	}
	return false
}

func (store *multiStore) pick(stores []Store, strategy MultiStorePutStrategy) (selected Store, err error) {
	if len(stores) == 0 {
		return nil, ErrStoreUnavailable
	}
	selected = stores[0]
	switch strategy {
	case Random:
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		selected = stores[r.Intn(len(stores))]
		for i := 0; i < 100 && !selected.IsWritable(); i++ {
			selected = stores[r.Intn(len(stores))]
		}
	case RoundRobin:
		rri := int(atomic.LoadInt64(&store.roundRobinIndex))
		for i := 1; i <= len(store.stores); i++ {
			j := (i + rri) % len(store.stores)
			if s := store.stores[j]; s.IsWritable() {
				selected = s
				atomic.StoreInt64(&store.roundRobinIndex, int64(j))
				break
			}
		}
	case MinCountFirst:
		for _, s := range stores[1:] {
			if s.IsWritable() && s.Count() < selected.Count() {
				selected = s
			}
		}
	case MinSizeFirst:
		for _, s := range stores[1:] {
			if s.IsWritable() && s.Size() < selected.Size() {
				selected = s
			}
		}
	}
	if selected.IsWritable() {
		return selected, nil
	}
	return nil, ErrStoreUnavailable
}
