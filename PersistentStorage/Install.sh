#!/bin/bash

#GO check
command -v go >/dev/null 2>&1 || { echo >&2 "GO is required but not installed.  Aborting."; exit 1; }
echo "Getting Go packages"
go get github.com/gorilla/mux
go get github.com/lib/pq

#Server Build
echo "Building Go Server"
mkdir -p bin;
cd Server/;
go build;
mv Server ../bin/PersistentStorageServer;
