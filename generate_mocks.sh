#!/usr/bin/env bash

BASEDIR=$(pwd)

echo "Making mocks..."

cd $BASEDIR/0chain.net/core || exit
mockery --output=../core/mocks --all

cd $BASEDIR/0chain.net/miner || exit
mockery --output=../miner/mocks --all

cd $BASEDIR/0chain.net/chaincore || exit
mockery --output=../chaincore/mocks --all

cd $BASEDIR/0chain.net/conductor || exit
mockery --output=../conductor/mocks --all

cd $BASEDIR/0chain.net/sharder || exit
mockery --output=../sharder/mocks --all

cd $BASEDIR/0chain.net/smartcontract || exit
mockery --output=../smartcontract/mocks --all

cd $BASEDIR/0chain.net/chaincore/chain/state || exit
mockery --name=StateContextI --output=../../../mocks --filename=state-context-i.go

echo "Mocks files are generated."