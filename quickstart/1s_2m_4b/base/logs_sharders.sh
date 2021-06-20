#!/bin/sh

. ./paths.sh

cd "$zChain_Root" || exit

#-------------------------------------------------

grep "$1" ./docker.local/sharder*/log/0chain.log
