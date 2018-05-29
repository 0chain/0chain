package persistencestore

import (
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
		session, err := getCluster().CreateSession()
	}
	return err
}
