#!/usr/bin/env bash

BASEDIR=$(pwd)

echo "Making mocks..."

 generate_mock() {
	OUTPUT=$1
	mockery --case underscore --output=$OUTPUT --all
}

cd $BASEDIR/code/go/0chain.net/core || exit
generate_mock "../core/mocks"

cd $BASEDIR/code/go/0chain.net/miner || exit
generate_mock "../miner/mocks"

cd $BASEDIR/code/go/0chain.net/chaincore || exit
generate_mock "../chaincore/mocks"

cd $BASEDIR/code/go/0chain.net/conductor || exit
generate_mock "../conductor/mocks"

cd $BASEDIR/code/go/0chain.net/sharder || exit
generate_mock "../sharder/mocks"

cd $BASEDIR/code/go/0chain.net/smartcontract || exit
generate_mock "../smartcontract/mocks"

cd $BASEDIR/code/go/0chain.net/chaincore/chain/state || exit
mockery --case underscore --name=StateContextI --output=../../../mocks

echo "Mocks files are generated."
