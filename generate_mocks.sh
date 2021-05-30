#!/usr/bin/env bash

BASEDIR=$(pwd)

echo "Making mocks..."

cd $BASEDIR/code/go/0chain.net/core || exit
mockery --output=../core/mocks --all --keeptree

cd $BASEDIR/code/go/0chain.net/miner || exit
mockery --output=../miner/mocks --all --keeptree

cd $BASEDIR/code/go/0chain.net/chaincore || exit
mockery --output=../chaincore/mocks --all --keeptree

cd $BASEDIR/code/go/0chain.net/conductor || exit
mockery --output=../conductor/mocks --all --keeptree

cd $BASEDIR/code/go/0chain.net/sharder || exit
mockery --output=../sharder/mocks --all --keeptree

cd $BASEDIR/code/go/0chain.net/smartcontract || exit
mockery --output=../smartcontract/mocks --all --keeptree

echo "Mocks files are generated."