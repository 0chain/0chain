#!/bin/bash
set -e

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
    CREATE extension ltree;
    CREATE DATABASE events_db;
    \connect events_db;
    CREATE USER zchain_user WITH ENCRYPTED PASSWORD 'zchian';
    GRANT ALL PRIVILEGES ON DATABASE events_db TO zchain_user;
EOSQL
