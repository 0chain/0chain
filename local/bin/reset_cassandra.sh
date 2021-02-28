#!/bin/sh

service cassandra stop
rm -rf /var/lib/cassandra/*
service cassandra start
#cqlsh -f "$(dirname "$0")"/../../docker.local/config/cassandra/init.cql