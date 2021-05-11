#!/usr/bin/env bash

BASEDIR=$(pwd)

echo "Making mocks..."

cd $BASEDIR/code/go/0chain.net/core/datastore || exit
mockery --name=Store --output=../../mocks/core/datastore --filename=store.go

cd $BASEDIR/code/go/0chain.net/core/persistencestore || exit
mockery --name=BatchI --output=../../mocks/core/persistencestore --filename=batch.go
mockery --name=IteratorI --output=../../mocks/core/persistencestore --filename=iterator.go
mockery --name=QueryI --output=../../mocks/core/persistencestore --filename=query.go
mockery --name=SessionI --output=../../mocks/core/persistencestore --filename=session.go

echo "Mocks files are generated."