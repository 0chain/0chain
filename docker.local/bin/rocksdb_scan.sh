#!/bin/sh
DB_DIR=$1;shift
/usr/local/Cellar/rocksdb/5.13.4/bin/rocksdb_ldb --db="$DB_DIR" "$*" scan

