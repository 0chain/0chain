package persistencestore

import (
	"github.com/gocql/gocql"
)

type SessionI interface {
	SetConsistency(cons gocql.Consistency)
	SetPageSize(n int)
	SetPrefetch(p float64)
	SetTrace(trace gocql.Tracer)
	Query(stmt string, values ...interface{}) *gocql.Query
	Bind(stmt string, b func(q *gocql.QueryInfo) ([]interface{}, error)) *gocql.Query
	Close()
	Closed() bool
	KeyspaceMetadata(keyspace string) (*gocql.KeyspaceMetadata, error)
	ExecuteBatch(batch *gocql.Batch) error
	ExecuteBatchCAS(batch *gocql.Batch, dest ...interface{}) (applied bool, iter *gocql.Iter, err error)
	MapExecuteBatchCAS(batch *gocql.Batch, dest map[string]interface{}) (applied bool, iter *gocql.Iter, err error)
	NewBatch(typ gocql.BatchType) *gocql.Batch
}

// Make sure that gocql.Session implements SessionI.
var _ SessionI = (*gocql.Session)(nil)
