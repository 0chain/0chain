package persistencestore

import (
	"context"
	"os"
	"time"

	"0chain.net/common"
	"0chain.net/datastore"
	"github.com/gocql/gocql"
)

var KeySpace = "zerochain"

// Session holds our connection to Cassandra
var Session *gocql.Session

func init() {
	var err error
	var cluster *gocql.ClusterConfig
	if os.Getenv("DOCKER") != "" {
		cluster = gocql.NewCluster("cassandra")
	} else {
		cluster = gocql.NewCluster("127.0.0.1")
	}
	cluster.Keyspace = KeySpace
	//TODO: Till we can have healthcheck in docker compose to work, we will keep waiting in the server code
	delay := time.Second
	for tries := 0; tries <= 40; tries++ {
		Session, err = cluster.CreateSession()
		if err != nil {
			time.Sleep(delay)
		} else {
			break
		}
	}
	if Session == nil {
		panic(err)
	}
}

/*GetConnection - returns a connection from the Pool
* Should always use right after getting the connection to avoid leaks
 * defer c.Close()
*/
func GetConnection() *gocql.Session {
	return Session
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

/*WithConnection takes a context and adds a connection value to it */
func WithEntityConnection(ctx context.Context, entityMetadata datastore.EntityMetadata) context.Context {
	return WithConnection(ctx)
}

func Close(ctx context.Context) {
	// TODO: Is this just a NOOP or anything required?
}
