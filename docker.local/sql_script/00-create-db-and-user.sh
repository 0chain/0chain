#!/bin/bash
set -e

DB_NAME=events_db
DB_USER=zchain_user
DB_PASSWORD=zchian

# Add extension
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
    CREATE extension ltree;
EOSQL

# Create the db if not exists
DB_EXISTS=$(psql -U "$POSTGRES_USER" -lqt | cut -d \| -f 1 | grep -w "$DB_NAME" || true);
if [ -z "$DB_EXISTS" ]; then
    psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" -c "CREATE DATABASE $DB_NAME";
fi

# Create the user if not exists, and give permissions
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
    \connect events_db;

    DO \$\$
    BEGIN
        IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = '$DB_USER') THEN
            CREATE USER $DB_USER WITH ENCRYPTED PASSWORD '$DB_PASSWORD';
        END IF;
    END
    \$\$;

    GRANT ALL PRIVILEGES ON DATABASE $DB_NAME TO $DB_USER;
EOSQL
