#!/bin/sh

service cassandra stop
rm -rf /var/lib/cassandra/*
service cassandra start