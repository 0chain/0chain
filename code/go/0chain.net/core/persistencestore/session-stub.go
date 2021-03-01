package persistencestore

import (
	"github.com/gocql/gocql"
)

// SessionMock implements SessionI interface and used when application starts with test mode.
type SessionMock struct{}

// Make sure SessionMock implements SessionI.
var _ SessionI = (*SessionMock)(nil)

func (s SessionMock) SetConsistency(_ gocql.Consistency) {}

func (s SessionMock) SetPageSize(_ int) {}

func (s SessionMock) SetPrefetch(_ float64) {}

func (s SessionMock) SetTrace(_ gocql.Tracer) {}

func (s SessionMock) Query(_ string, _ ...interface{}) *gocql.Query {
	return &gocql.Query{}
}

func (s SessionMock) Bind(_ string, _ func(q *gocql.QueryInfo) ([]interface{}, error)) *gocql.Query {
	return nil
}

func (s SessionMock) Closed() bool { return false }

func (s SessionMock) KeyspaceMetadata(_ string) (*gocql.KeyspaceMetadata, error) {
	return nil, nil
}

func (s SessionMock) ExecuteBatch(_ *gocql.Batch) error { return nil }

func (s SessionMock) ExecuteBatchCAS(_ *gocql.Batch,
	_ ...interface{}) (applied bool, iter *gocql.Iter, err error) {
	return
}

func (s SessionMock) MapExecuteBatchCAS(_ *gocql.Batch,
	_ map[string]interface{}) (applied bool, iter *gocql.Iter, err error) {
	return
}

func (s SessionMock) NewBatch(_ gocql.BatchType) *gocql.Batch { return nil }

func (s SessionMock) Close() {}
