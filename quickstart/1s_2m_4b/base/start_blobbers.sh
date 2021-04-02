#!/bin/bash

. ./paths.sh

cd $zBlober_Root

#----------------------------------------------

cd ./docker.local/blobber1

PWD=`pwd`
BLOBBER_DIR=`basename $PWD`
BLOBBER_ID=`echo my directory $BLOBBER_DIR | sed -e 's/.*\(.\)$/\1/'`
echo Starting blobber$BLOBBER_ID ...
BLOBBER=$BLOBBER_ID docker-compose -p blobber$BLOBBER_ID -f ../b0docker-compose.yml up -d &



cd ../blobber2

PWD=`pwd`
BLOBBER_DIR=`basename $PWD`
BLOBBER_ID=`echo my directory $BLOBBER_DIR | sed -e 's/.*\(.\)$/\1/'`
echo Starting blobber$BLOBBER_ID ...
BLOBBER=$BLOBBER_ID docker-compose -p blobber$BLOBBER_ID -f ../b0docker-compose.yml up -d &

cd ../blobber3

PWD=`pwd`
BLOBBER_DIR=`basename $PWD`
BLOBBER_ID=`echo my directory $BLOBBER_DIR | sed -e 's/.*\(.\)$/\1/'`
echo Starting blobber$BLOBBER_ID ...
BLOBBER=$BLOBBER_ID docker-compose -p blobber$BLOBBER_ID -f ../b0docker-compose.yml up -d &


cd ../blobber4

PWD=`pwd`
BLOBBER_DIR=`basename $PWD`
BLOBBER_ID=`echo my directory $BLOBBER_DIR | sed -e 's/.*\(.\)$/\1/'`
echo Starting blobber$BLOBBER_ID ...
BLOBBER=$BLOBBER_ID docker-compose -p blobber$BLOBBER_ID -f ../b0docker-compose.yml up -d &