#!/bin/bash

. ./paths.sh


#-------------------------------------------------------

./stopp_miners.sh

#-------------------------------------------------------

cd $zChain_Root

sleep 1

sudo rm -rf ./docker.local/miner*/log/*

sleep 

#-------------------------------------------------------

cd $zWorkflows_Base

./start_miners.sh