#!/bin/sh

. ./paths.sh
#-------------------------------------------------
./stop_blobbers.sh

#-------------------------------------------------
cd "$zBlober_Root" || exit
sudo rm -rf ./docker.local/blobber*/log/*

sleep 1
#-------------------------------------------------
cd "$zWorkflows_Base" || exit
./start_blobbers.sh