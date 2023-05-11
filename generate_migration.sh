#!/bin/bash
ts=$(date +%s)
fname=$1

if [ -z "$fname" ]; then
    echo "Usage: $0 <filename>"
    exit 1
fi


# Write boilerplate of the migration file
echo "-- +goose Up
-- +goose StatementBegin

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- +goose StatementEnd" > "code/go/0chain.net/smartcontract/dbs/goose/migrations/""$ts""_$fname.sql"