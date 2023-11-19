package blockdb

import (
	"context"
	"errors"
	"io"
)

var ErrKeyNotFound = errors.New("key not found")

// Key - type for the record's identifier
type Key string

// SerDe - serialize/deserialize data
type SerDe interface {
	Encode(writer io.Writer) error
	Decode(reader io.Reader) error
}

// Record - an interface to read and write a record
type Record interface {
	GetKey() Key
	SerDe
}

// RecordProvider - a factory to create a new record object
type RecordProvider interface {
	NewRecord() Record
}

// Index - the database index interface
type Index interface {
	SetOffset(key Key, offset int64) error
	GetOffset(key Key) (int64, error)
	GetKeys() []Key
	SerDe
}

// DBIteratorHandler - an interator handler that handles each record in the db
type DBIteratorHandler func(ctx context.Context, record Record) error

// DBHeader - an interface to read and write the db header
type DBHeader interface {
	SerDe
}

// Database - a database interface for an immutable database of fixed set of records
type Database interface {
	Create() error
	Open() error
	Save() error
	Close() error
	Delete() error
	SetDBHeader(dbheader DBHeader)
	SetIndex(index Index)

	ReadAll(rp RecordProvider) ([]Record, error)
	Read(key Key, record Record) error
	WriteData(record Record) error
	Iterate(ctx context.Context, handler DBIteratorHandler, rp RecordProvider) error
}
