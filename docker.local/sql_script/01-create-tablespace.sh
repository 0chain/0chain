#!/bin/bash
set -e

mkdir -p $SLOW_TABLESPACE_PATH

TABLESPACE_EXISTS=$(psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" -tAc "SELECT 1 FROM pg_tablespace WHERE spcname = '$SLOW_TABLESPACE'")

if [ -z "$TABLESPACE_EXISTS" ]; then
    psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" -c "create tablespace $SLOW_TABLESPACE location '$SLOW_TABLESPACE_PATH';";
fi

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
    ALTER TABLESPACE $SLOW_TABLESPACE OWNER TO zchain_user;
EOSQL
