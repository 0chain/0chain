#!/bin/sh

. ./paths.sh


#-------------------------------------------------------

./stopp_miners.sh

#-------------------------------------------------------

cd "$zChain_Root" || exit

sleep 1

sudo rm -rf ./docker.local/miner*/log/*

sleep 

#-------------------------------------------------------

cd "$zWorkflows_Base" || exit

./start_miners.sh