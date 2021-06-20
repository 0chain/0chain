#!/bin/sh

. ./paths.sh

cd "$zBlober_Root" || exit


#-------------------------------------------------

cd ./docker.local/blobber1 || exit

#../bin/blobber.stop_bls.sh

PWD=`pwd`
BLOBBER_DIR=`basename "$PWD"`
BLOBBER_ID=`echo my directory "$BLOBBER_DIR" | sed -e 's/.*\(.\)$/\1/'`
echo Stopping blobber"$BLOBBER_ID" ...
BLOBBER=$BLOBBER_ID docker-compose -p blobber"$BLOBBER_ID" -f ../b0docker-compose.yml stop



cd ../blobber2 || exit


PWD=`pwd`
BLOBBER_DIR=`basename "$PWD"`
BLOBBER_ID=`echo my directory "$BLOBBER_DIR" | sed -e 's/.*\(.\)$/\1/'`
echo Stopping blobber"$BLOBBER_ID" ...
BLOBBER=$BLOBBER_ID docker-compose -p blobber"$BLOBBER_ID" -f ../b0docker-compose.yml stop

cd ../blobber3 || exit


PWD=`pwd`
BLOBBER_DIR=`basename "$PWD"`
BLOBBER_ID=`echo my directory "$BLOBBER_DIR" | sed -e 's/.*\(.\)$/\1/'`
echo Stopping blobber"$BLOBBER_ID" ...
BLOBBER=$BLOBBER_ID docker-compose -p blobber"$BLOBBER_ID" -f ../b0docker-compose.yml stop


cd ../blobber4 || exit


PWD=`pwd`
BLOBBER_DIR=`basename "$PWD"`
BLOBBER_ID=`echo my directory "$BLOBBER_DIR" | sed -e 's/.*\(.\)$/\1/'`
echo Stopping blobber"$BLOBBER_ID" ...
BLOBBER=$BLOBBER_ID docker-compose -p blobber"$BLOBBER_ID" -f ../b0docker-compose.yml stop
