#!/bin/sh
PWD=`pwd`
NODE_DIR=`basename $PWD`
NODE_ID=`echo $NODE_DIR | sed -e 's/.*\(.\)$/\1/'`

SERVICE=$1; shift
CMD=$1; shift

echo $NODE_DIR: running $SERVICE $CMD $*

SHARDER=$SHARDER_ID docker-compose -p $NODE_DIR -f ../build.sharder/docker-compose.yml exec $SERVICE $CMD $*
