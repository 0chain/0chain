#!/bin/sh

. ./paths.sh

cd "$zChain_Root" || exit

#-------------------------------------------------

grep "$1" ./docker.local/miner*/log/0chain.log
