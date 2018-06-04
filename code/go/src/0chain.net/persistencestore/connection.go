package persistencestore

import (
	"context"
	"fmt"

	"0chain.net/common"
	"0chain.net/datastore"
	"github.com/gocql/gocql"
)

// Session holds our connection to Cassandra
var Session *gocql.Session

func init() {
	var err error

	cluster := gocql.NewCluster("127.0.0.1")
	cluster.Keyspace = "ochaincluster"
	Session, err = cluster.CreateSession()
	if err != nil {
		panic(err)
	}
	fmt.Println("cassandra init done")
}

// /*GetConnection - returns a connection from the Pool
// * Should always use right after getting the connection to avoid leaks
// * defer c.Close()
//  */
func GetConnection() *gocql.Session {
	fmt.Println("in cassandra")
	return Session
}

//
// /*CONNECTION - key used to get the connection object from the context */
const CONNECTION common.ContextKey = "pconnection"

//
/*WithConnection takes a context and adds a connection value to it */
func WithConnection(ctx context.Context, entityMetadata datastore.EntityMetadata) context.Context {
	return context.WithValue(ctx, CONNECTION, GetConnection())
}

// /*GetCon returns a connection stored in the context which got created via WithConnection */
func GetCon(ctx context.Context) *gocql.Session {
	fmt.Println("I am at cassandra")
	if ctx == nil {
		return GetConnection()
	}
	return ctx.Value(CONNECTION).(*gocql.Session)
}
