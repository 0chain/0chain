/0chain/cassandra/wait-for-it.sh -t 0 cassandra:9042 -- echo "CASSANDRA Node1 started"

cqlsh -f /0chain/cassandra/init.cql cassandra

echo "### CASSANDRA INITIALISED! ###"
