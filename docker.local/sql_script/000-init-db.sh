#!/bin/bash
set -e

mkdir -p $SLOW_TABLESPACE_PATH

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
create tablespace $SLOW_TABLESPACE location '$SLOW_TABLESPACE_PATH';
EOSQL
