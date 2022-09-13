#!/usr/bin/env bash

BASEDIR=$(pwd)

echo "Making mocks..."

 generate_mock() {
	OUTPUT=$1
	mockery --case underscore --output=$OUTPUT --all
}

cd $BASEDIR/code/go/0chain.net/core || exit
rm -rf ../core/mocks
generate_mock "../core/mocks"

cd $BASEDIR/code/go/0chain.net/miner || exit
rm -rf ../miner/mocks
generate_mock "../miner/mocks"

cd $BASEDIR/code/go/0chain.net/chaincore || exit
rm -rf ../chaincore/mocks
generate_mock "../chaincore/mocks"

cd $BASEDIR/code/go/0chain.net/chaincore/chain/state || exit
rm -rf ../state/mocks
generate_mock "../state/mocks"

cd $BASEDIR/code/go/0chain.net/conductor || exit
rm -rf ../conductor/mocks
generate_mock "../conductor/mocks"

cd $BASEDIR/code/go/0chain.net/sharder || exit
rm -rf ../sharder/mocks
generate_mock "../sharder/mocks"

cd $BASEDIR/code/go/0chain.net/smartcontract || exit
rm -rf ../smartcontract/mocks
generate_mock "../smartcontract/mocks"

cd $BASEDIR/code/go/0chain.net/smartcontract/benchmark || exit
rm -rf ../benchmark/mocks
generate_mock "../benchmark/mocks"

cd $BASEDIR/code/go/0chain.net || exit
go generate -run="mockery" ./...


echo "generating mocks from common repo"
version=$(grep "github.com/0chain/common" $BASEDIR/code/go/0chain.net/go.mod | awk '{print $3}') # get the version from go.mod
OCHAIN_COMMON_VERSION=${version##*-}
mkdir temp
git clone --depth 1 --branch $OCHAIN_COMMON_VERSION https://github.com/0chain/common.git $BASEDIR/temp
cd $BASEDIR/temp
generate_mock "$BASEDIR/code/go/0chain.net/core/mocks"
cd $BASEDIR
rm -rf temp
echo "Mocks files are generated."
