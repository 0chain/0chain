#!/bin/bash

. ./paths.sh

cd $zWallet_Root

#---------------------------------------------------

rm ~/.zcn/wallet.json

$zWallet_Root/zwallet faucet --methodName pour --input "{Pay day}"
$zWallet_Root/zwallet faucet --methodName pour --input "{Pay day}"
$zWallet_Root/zwallet faucet --methodName pour --input "{Pay day}"
$zWallet_Root/zwallet faucet --methodName pour --input "{Pay day}"
$zWallet_Root/zwallet faucet --methodName pour --input "{Pay day}"
$zWallet_Root/zwallet faucet --methodName pour --input "{Pay day}"
$zWallet_Root/zwallet faucet --methodName pour --input "{Pay day}"
$zWallet_Root/zwallet faucet --methodName pour --input "{Pay day}"


$zWallet_Root/zwallet getbalance

cat ~/.zcn/wallet.json