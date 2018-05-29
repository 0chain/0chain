package persistencestore

import (
	"context"

	"0chain.net/common"
	"github.com/gocql/gocql"
)

// Session holds our connection to Cassandra
var session *gocql.Session

func getCluster() *gocql.ClusterConfig {
	cluster := gocql.NewCluster("127.0.0.1")
	cluster.Keyspace = "ochaincluster"
	cluster.Consistency = gocql.One
	cluster.Port = 9042 // default port
	cluster.NumConns = 1
	return cluster
}

func initSession() error {
	var err error
	if session == nil || session.Closed() {
		session, err = getCluster().CreateSession()
	}
	return err
}

/*GetConnection - returns a connection from the Pool
* Should always use right after getting the connection to avoid leaks
* defer c.Close()
 */
func GetConnection() *gocql.Session {
	return session
}

/*CONNECTION - key used to get the connection object from the context */
const CONNECTION common.ContextKey = "pconnection"

/*WithConnection takes a context and adds a connection value to it */
func WithConnection(ctx context.Context) context.Context {
	return context.WithValue(ctx, CONNECTION, GetConnection())
}

/*GetCon returns a connection stored in the context which got created via WithConnection */
func GetCon(ctx context.Context) *gocql.Session {
	if ctx == nil {
		return GetConnection()
	}
	return ctx.Value(CONNECTION).(*gocql.Session)
}
