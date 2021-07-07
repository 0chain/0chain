#!/usr/bin/env bash
# This files needs to be run from the ./code/go directory
# cd ./code/go && ../../generate_mocks.sh

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

echo "Mocks files are generated."