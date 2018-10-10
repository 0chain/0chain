package persistencestore

import (
	"context"
	"os"
	"time"

	"0chain.net/common"
	"0chain.net/datastore"
	. "0chain.net/logging"
	"github.com/gocql/gocql"
	"go.uber.org/zap"
)

//KeySpace - the keyspace usef for the 0chain data
var KeySpace = "zerochain"

//ClusterName - name of the cluster used for cassandra or compatible service
var ClusterName = "cassandra"

func init() {
	cname := os.Getenv("CASSANDRA_CLUSTER")
	if cname != "" {
		ClusterName = cname
	}
}

// Session holds our connection to Cassandra
var Session *gocql.Session

/*InitSession - initialize a storage session */
func InitSession() {
	err := initSession(time.Second, 0)
	if Session == nil {
		panic(err)
	}
}

func initSession(delay time.Duration, maxTries int) error {
	var err error
	var cluster *gocql.ClusterConfig
	if os.Getenv("DOCKER") != "" {
		cluster = gocql.NewCluster(ClusterName)
	} else {
		cluster = gocql.NewCluster("127.0.0.1")
	}

	// Setting the following for now because of https://github.com/gocql/gocql/issues/1200
	cluster.WriteCoalesceWaitTime = 0

	// This reduces the time to create the session from 9+ seconds to 5 seconds when running the tests.
	//cluster.DisableInitialHostLookup = true

	cluster.Keyspace = KeySpace
	// We need to keep waiting till whatever time it takes for cassandra to come up and running that includes data operations which takes longer with growing data
	for tries := 0; maxTries <= 0 || tries <= maxTries; tries++ {
		start := time.Now()
		Session, err = cluster.CreateSession()
		Logger.Info("time to creation cassandra session", zap.Any("duration", time.Since(start)))
		if err != nil {
			Logger.Error("error creating session", zap.Any("retry", tries), zap.Error(err))
			time.Sleep(delay)
		} else {
			return nil
		}
	}
	return err
}

/*GetConnection - returns a connection from the Pool
* Should always use right after getting the connection to avoid leaks
 * defer c.Close()
*/
func GetConnection() *gocql.Session {
	if Session == nil {
		InitSession()
	}
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

/*WithEntityConnection takes a context and adds a connection value to it */
func WithEntityConnection(ctx context.Context, entityMetadata datastore.EntityMetadata) context.Context {
	return WithConnection(ctx)
}

/*Close - close all the connections in the context */
func Close(ctx context.Context) {
	// TODO: Is this just a NOOP or anything required?
}
