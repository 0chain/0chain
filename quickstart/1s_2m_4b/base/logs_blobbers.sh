#!/bin/bash

. ./paths.sh

cd $zBlober_Root

#-------------------------------------------------

echo "BLOBBERS LOGS"

grep $1 ./docker.local/blobber*/log/0chainBlobber.log

echo "VALIDATORS LOGS"

grep $1 ./docker.local/blobber*/log/0chainBlobber.log