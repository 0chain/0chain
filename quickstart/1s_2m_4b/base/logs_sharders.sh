#!/bin/bash

. ./paths.sh

cd $zChain_Root

#-------------------------------------------------

grep $1 ./docker.local/sharder*/log/0chain.log