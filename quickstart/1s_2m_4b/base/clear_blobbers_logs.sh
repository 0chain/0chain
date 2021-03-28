#!/bin/bash

. ./paths.sh

#-------------------------------------------------


./stop_blobbers.sh


#-------------------------------------------------

cd $zBlober_Root

sudo rm -rf ./docker.local/blobber*/log/*


sleep 1
#-------------------------------------------------

cd $zWorkflows_Base

./start_blobbers.sh