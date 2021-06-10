package persistencestore

import (
	"errors"

	"github.com/gocql/gocql"
)

type (
	SessionI interface {
		Query(string, ...interface{}) QueryI
		NewBatch(gocql.BatchType) BatchI
		ExecuteBatch(BatchI) error
		Close()
	}

	session struct {
		*gocql.Session
	}
)

// Make sure that session implements SessionI.
var _ SessionI = (*session)(nil)

// NewSession instantiates a new session.
func NewSession(s *gocql.Session) SessionI {
	return &session{Session: s}
}

// Query generates a new query object for interacting with the database.
// Query is automatically prepared if it has not previously been executed.
func (s *session) Query(stmt string, values ...interface{}) QueryI {
	q := s.Session.Query(stmt, values...)
	return &query{Query: q}
}

// NewBatch creates a new batch operation using defaults defined in the cluster.
func (s *session) NewBatch(typ gocql.BatchType) BatchI {
	b := s.Session.NewBatch(typ)
	return &batch{Batch: b}
}

// ExecuteBatch executes a batch operation and returns nil if successful
// otherwise an error is returned describing the failure.
// Note: works only with default batch implementation.
func (s *session) ExecuteBatch(b BatchI) error {
	bat, ok := b.(*batch)
	if !ok {
		return errors.New("unknown batch")
	}

	return s.Session.ExecuteBatch(bat.Batch)
}

type (
	QueryI interface {
		Bind(...interface{}) QueryI
		Exec() error
		Iter() IteratorI
		Scan(...interface{}) error
	}

	query struct {
		*gocql.Query
	}
)

// Make sure that query implements QueryI.
var _ QueryI = (*query)(nil)

// NewQuery instantiates a new query.
func NewQuery(q *gocql.Query) QueryI {
	return &query{Query: q}
}

// Iter executes the query and returns an iterator capable of iterating over all results.
func (q *query) Iter() IteratorI {
	return q.Query.Iter()
}

// Bind sets query arguments of query.
// This can also be used to rebind new query arguments to an existing query instance.
func (q *query) Bind(v ...interface{}) QueryI {
	return NewQuery(q.Query.Bind(v))
}

type (
	IteratorI interface {
		Scan(...interface{}) bool
		Close() error
	}

	iterator struct {
		*gocql.Iter
	}
)

// Make sure that iterator implements IteratorI.
var _ IteratorI = (*iterator)(nil)

type (
	BatchI interface {
		Query(string, ...interface{})
	}

	batch struct {
		*gocql.Batch
	}
)

// Make sure that batch implements BatchI.
var _ BatchI = (*batch)(nil)
