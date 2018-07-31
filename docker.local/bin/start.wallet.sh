#!/bin/sh
PWD=`pwd`
WALLET_DIR=`basename $PWD`
WALLET_ID=`echo $WALLET_DIR | sed -e 's/.*\(.\)$/\1/'`


echo Starting wallet$WALLET_ID ...

WALLET=$WALLET_ID docker-compose -p wallet$WALLET_ID -f ../build.wallet/docker-compose.yml up