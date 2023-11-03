#!/bin/bash
set -e

mkdir -p $SLOW_TABLESPACE_PATH

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
    create tablespace $SLOW_TABLESPACE location '$SLOW_TABLESPACE_PATH';
    alter tablespace $SLOW_TABLESPACE set ( seq_page_cost=10, random_page_cost=10 );
EOSQL
