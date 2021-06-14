#!/bin/sh
set -e

go_mod=$1

echo Building dependencies from "$go_mod"

deps="$( \
        < "$go_mod" \
        grep -F ' v' | \
        grep -F -v 0chain.net | \
        grep -F -v pbc | \
        sed -E 's/^\t([a-zA-Z0-9./-]+) ((v[0-9.]+(\+incompatible)?)($| ))?(v[0-9\.]+-[0-9]+-([0-9a-f]+))?.*$/\1@\3\7/' \
      )"
#echo Deps are "$deps"

cd "$(dirname "$go_mod")"
go get -v -tags bn256 $deps
