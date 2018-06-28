#!/bin/bash
chmod +x /0chain/cassandra/wait-for-it.sh

/0chain/cassandra/wait-for-it.sh -t 0 cassandra:9042 -- echo "CASSANDRA Node started"


cqlsh -f /0chain/cassandra/init.cql cassandra

echo "### CASSANDRA INITIALISED! ###"
