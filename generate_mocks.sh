#!/usr/bin/env bash

BASEDIR=$(pwd)

echo "Making mocks..."

cd $BASEDIR/code/go/0chain.net/core/datastore || exit
mockery --name=Store --output=../../mocks --filename=store.go

cd $BASEDIR/code/go/0chain.net/core/persistencestore || exit
mockery --name=BatchI --output=../../mocks --filename=batch.go
mockery --name=IteratorI --output=../../mocks --filename=iterator.go
mockery --name=QueryI --output=../../mocks --filename=query.go
mockery --name=SessionI --output=../../mocks --filename=session.go

cd $BASEDIR/code/go/0chain.net/core/util || exit
mockery --name=Serializable --output=../../mocks --filename=serializable.go

cd $BASEDIR/code/go/0chain.net/smartcontract || exit
mockery --output=../mocks/smartcontract --all

echo "Mocks files are generated."