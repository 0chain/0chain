#!/bin/bash
set -e

chown postgres:postgres $SLOW_TABLESPACE_PATH
chmod g+rwx $SLOW_TABLESPACE_PATH

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
create tablespace $SLOW_TABLESPACE owner $POSTGRES_USER location $SLOW_TABLESPACE_PATH;
EOSQL
