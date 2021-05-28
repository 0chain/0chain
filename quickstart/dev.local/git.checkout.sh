#!/bin/bash

#skip 0chain/0chain

echo "checkout 0dns"
[ -d $zDNS ] || git clone git@github.com:0chain/0dns.git $zDNS

echo "checkout blobber"
[ -d $zBlobber ] || git clone git@github.com:0chain/blobber.git $zBlobber

echo "checkout 0miner"
[ -d $zMiner ] || git clone git@github.com:0chain/0miner.git $zMiner

echo "checkout zboxcli"
[ -d $zBox ] || git clone git@github.com:0chain/zboxcli.git $zBox

echo "checkout zwalletcli"
[ -d $zWallet ] || git clone git@github.com:0chain/zwalletcli.git $zWallet
