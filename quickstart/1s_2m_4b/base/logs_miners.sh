#!/bin/bash

. ./paths.sh

cd $zChain_Root

#-------------------------------------------------

grep $1 ./docker.local/miner*/log/0chain.log
