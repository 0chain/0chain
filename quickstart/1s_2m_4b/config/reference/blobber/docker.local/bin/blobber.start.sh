#!/bin/sh
PWD=`pwd`
BLOBBER_DIR=`basename $PWD`
BLOBBER_ID=`echo my directory $BLOBBER_DIR | sed -e 's/.*\(.\)$/\1/'`


echo Starting blobber$BLOBBER_ID ...

# echo blobber$i

BLOBBER=$BLOBBER_ID docker-compose -p blobber$BLOBBER_ID -f ../docker-compose.yml up
