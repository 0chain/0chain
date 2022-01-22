#!/bin/bash

set -e

psql=( psql --username "$POSTGRES_USER" --port "$POSTGRES_PORT" --host "$POSTGRES_HOST" )

until pg_isready -h $POSTGRES_HOST
do
	echo "Sleep 1s and try again..."
	sleep 1
done

export PGPASSWORD="$POSTGRES_PASSWORD"

for f in /blobber/sql/*; do
	case "$f" in
		*.sh)     echo "$0: running $f"; . "$f" ;;
		*.sql)    echo "$0: running $f"; "${psql[@]}" -f "$f"; echo ;;
		*.sql.gz) echo "$0: running $f"; gunzip -c "$f" | "${psql[@]}"; echo ;;
		*)        echo "$0: ignoring $f" ;;
	esac
	echo
done
