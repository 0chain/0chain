#!/bin/bash

. ./paths.sh

cd $zWallet_Root

#---------------------------------------------------

rm ~/.zcn/wallet.json

./zwallet faucet --methodName pour --input "{Pay day}"
./zwallet faucet --methodName pour --input "{Pay day}"
./zwallet faucet --methodName pour --input "{Pay day}"
./zwallet faucet --methodName pour --input "{Pay day}"
./zwallet faucet --methodName pour --input "{Pay day}"
./zwallet faucet --methodName pour --input "{Pay day}"
./zwallet faucet --methodName pour --input "{Pay day}"
./zwallet faucet --methodName pour --input "{Pay day}"


./zwallet getbalance

cat ~/.zcn/wallet.json