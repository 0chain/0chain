package store

import (
	"errors"
	"io"
)

var (
	ErrKeyNotFound = errors.New("key not found")
)

type Store interface {
	Put(key, value []byte) (err error)
	Get(key []byte) (value []byte, err error)
	PutReader(key []byte, r io.Reader) (err error)
	GetReader(key []byte) (r io.ReadCloser, err error)
	Delete(key []byte) (err error)
	Size() int64
	Count() int64
	IsExist(key []byte) bool
	IsWritable() bool
}
