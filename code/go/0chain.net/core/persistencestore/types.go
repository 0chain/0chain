package persistencestore

import (
	"context"
	"errors"
	"github.com/gocql/gocql"
)

type (
	SessionI interface {
		SetConsistency(cons gocql.Consistency)
		SetPageSize(n int)
		SetPrefetch(p float64)
		SetTrace(trace gocql.Tracer)
		Query(stmt string, values ...interface{}) QueryI
		Bind(stmt string, b func(q *gocql.QueryInfo) ([]interface{}, error)) QueryI
		Close()
		Closed() bool
		KeyspaceMetadata(keyspace string) (*gocql.KeyspaceMetadata, error)
		ExecuteBatch(batch BatchI) error
		ExecuteBatchCAS(batch *gocql.Batch, dest ...interface{}) (applied bool, iter *gocql.Iter, err error)
		MapExecuteBatchCAS(batch *gocql.Batch, dest map[string]interface{}) (applied bool, iter *gocql.Iter, err error)
		NewBatch(gocql.BatchType) BatchI
	}

	session struct {
		*gocql.Session
	}

	QueryI interface {
		Statement() string
		String() string
		Attempts() int
		AddAttempts(i int, host *gocql.HostInfo)
		Latency() int64
		AddLatency(l int64, host *gocql.HostInfo)
		Consistency(c gocql.Consistency) *gocql.Query
		GetConsistency() gocql.Consistency
		SetConsistency(c gocql.Consistency)
		CustomPayload(customPayload map[string][]byte) *gocql.Query
		Context() context.Context
		Trace(trace gocql.Tracer) *gocql.Query
		Observer(observer gocql.QueryObserver) *gocql.Query
		PageSize(n int) *gocql.Query
		DefaultTimestamp(enable bool) *gocql.Query
		WithTimestamp(timestamp int64) *gocql.Query
		RoutingKey(routingKey []byte) *gocql.Query
		WithContext(ctx context.Context) *gocql.Query
		Cancel()
		Keyspace() string
		GetRoutingKey() ([]byte, error)
		Prefetch(p float64) *gocql.Query
		RetryPolicy(r gocql.RetryPolicy) *gocql.Query
		SetSpeculativeExecutionPolicy(sp gocql.SpeculativeExecutionPolicy) *gocql.Query
		IsIdempotent() bool
		Idempotent(value bool) *gocql.Query
		Bind(v ...interface{}) *gocql.Query
		SerialConsistency(cons gocql.SerialConsistency) *gocql.Query
		PageState(state []byte) *gocql.Query
		NoSkipMetadata() *gocql.Query
		Exec() error
		Iter() IteratorI
		MapScan(m map[string]interface{}) error
		Scan(dest ...interface{}) error
		ScanCAS(dest ...interface{}) (applied bool, err error)
		MapScanCAS(dest map[string]interface{}) (applied bool, err error)
		Release()
	}

	query struct {
		*gocql.Query
	}

	IteratorI interface {
		RowData() (gocql.RowData, error)
		SliceMap() ([]map[string]interface{}, error)
		MapScan(m map[string]interface{}) bool
		Host() *gocql.HostInfo
		Columns() []gocql.ColumnInfo
		Scanner() gocql.Scanner
		Scan(dest ...interface{}) bool
		GetCustomPayload() map[string][]byte
		Warnings() []string
		Close() error
		WillSwitchPage() bool
		PageState() []byte
		NumRows() int
	}

	iterator struct {
		*gocql.Iter
	}

	BatchI interface {
		Observer(observer gocql.BatchObserver) *gocql.Batch
		Keyspace() string
		Attempts() int
		AddAttempts(i int, host *gocql.HostInfo)
		Latency() int64
		AddLatency(l int64, host *gocql.HostInfo)
		GetConsistency() gocql.Consistency
		SetConsistency(c gocql.Consistency)
		Context() context.Context
		IsIdempotent() bool
		SpeculativeExecutionPolicy(sp gocql.SpeculativeExecutionPolicy) *gocql.Batch
		Query(stmt string, args ...interface{})
		Bind(stmt string, bind func(q *gocql.QueryInfo) ([]interface{}, error))
		RetryPolicy(r gocql.RetryPolicy) *gocql.Batch
		WithContext(ctx context.Context) *gocql.Batch
		Cancel()
		Size() int
		SerialConsistency(cons gocql.SerialConsistency) *gocql.Batch
		DefaultTimestamp(enable bool) *gocql.Batch
		WithTimestamp(timestamp int64) *gocql.Batch
		GetRoutingKey() ([]byte, error)
	}

	batch struct {
		*gocql.Batch
	}
)

var (
	// Make sure that session implements SessionI.
	_ SessionI = (*session)(nil)

	// Make sure that query implements QueryI.
	_ QueryI = (*query)(nil)

	// Make sure that iterator implements IteratorI.
	_ IteratorI = (*iterator)(nil)

	// Make sure that batch implements BatchI.
	_ BatchI = (*batch)(nil)
)

func (s *session) Query(stmt string, values ...interface{}) QueryI {
	q := s.Session.Query(stmt, values...)
	return &query{Query: q}
}

func (s *session) Bind(stmt string, b func(q *gocql.QueryInfo) ([]interface{}, error)) QueryI {
	q := s.Session.Bind(stmt, b)
	return &query{Query: q}
}

func (s *session) NewBatch(typ gocql.BatchType) BatchI {
	b := s.Session.NewBatch(typ)
	return &batch{Batch: b}
}

// ExecuteBatch executes a batch operation and returns nil if successful
// otherwise an error is returned describing the failure.
// Note: works only with batch implementation.
func (s *session) ExecuteBatch(b BatchI) error {
	bat, ok := b.(batch)
	if !ok {
		return errors.New("unknown batch")
	}

	return s.Session.ExecuteBatch(bat.Batch)
}

func (q *query) Iter() IteratorI {
	return q.Query.Iter()
}
