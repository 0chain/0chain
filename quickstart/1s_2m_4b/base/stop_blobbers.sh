#!/bin/bash

. ./paths.sh

cd $zBlober_Root


#-------------------------------------------------

cd ./docker.local/blobber1

#../bin/blobber.stop_bls.sh

PWD=`pwd`
BLOBBER_DIR=`basename $PWD`
BLOBBER_ID=`echo my directory $BLOBBER_DIR | sed -e 's/.*\(.\)$/\1/'`
echo Stopping blobber$BLOBBER_ID ...
BLOBBER=$BLOBBER_ID docker-compose -p blobber$BLOBBER_ID -f ../b0docker-compose.yml stop



cd ../blobber2


PWD=`pwd`
BLOBBER_DIR=`basename $PWD`
BLOBBER_ID=`echo my directory $BLOBBER_DIR | sed -e 's/.*\(.\)$/\1/'`
echo Stopping blobber$BLOBBER_ID ...
BLOBBER=$BLOBBER_ID docker-compose -p blobber$BLOBBER_ID -f ../b0docker-compose.yml stop

cd ../blobber3


PWD=`pwd`
BLOBBER_DIR=`basename $PWD`
BLOBBER_ID=`echo my directory $BLOBBER_DIR | sed -e 's/.*\(.\)$/\1/'`
echo Stopping blobber$BLOBBER_ID ...
BLOBBER=$BLOBBER_ID docker-compose -p blobber$BLOBBER_ID -f ../b0docker-compose.yml stop


cd ../blobber4


PWD=`pwd`
BLOBBER_DIR=`basename $PWD`
BLOBBER_ID=`echo my directory $BLOBBER_DIR | sed -e 's/.*\(.\)$/\1/'`
echo Stopping blobber$BLOBBER_ID ...
BLOBBER=$BLOBBER_ID docker-compose -p blobber$BLOBBER_ID -f ../b0docker-compose.yml stop