#!/usr/bin/env bash

BASEDIR=$(pwd)

docker pull vektra/mockery
alias mockery='docker run -v "$PWD":/src -w /src vektra/mockery'

echo "Making mocks..."

cd $BASEDIR/code/go/0chain.net/core || exit
mockery --output=../core/mocks --all

cd $BASEDIR/code/go/0chain.net/miner || exit
mockery --output=../miner/mocks --all

cd $BASEDIR/code/go/0chain.net/chaincore || exit
mockery --output=../chaincore/mocks --all

cd $BASEDIR/code/go/0chain.net/conductor || exit
mockery --output=../conductor/mocks --all

cd $BASEDIR/code/go/0chain.net/sharder || exit
mockery --output=../sharder/mocks --all

cd $BASEDIR/code/go/0chain.net/smartcontract || exit
mockery --output=../smartcontract/mocks --all

echo "Mocks files are generated."