#!/bin/bash

# docker.local/bin/start.miner

. ./paths.sh

cd $zChain_Root


#----------------------------------------------


cd ./docker.local/miner1


PWD=`pwd`
MINER_DIR=`basename $PWD`
MINER_ID=`echo $MINER_DIR | sed -e 's/.*\(.\)$/\1/'`
echo Starting miner$MINER_ID ...
MINER=$MINER_ID docker-compose -p miner$MINER_ID -f ../build.miner/b0docker-compose.yml up &

cd ../miner2

PWD=`pwd`
MINER_DIR=`basename $PWD`
MINER_ID=`echo $MINER_DIR | sed -e 's/.*\(.\)$/\1/'`
echo Starting miner$MINER_ID ...
MINER=$MINER_ID docker-compose -p miner$MINER_ID -f ../build.miner/b0docker-compose.yml up &


#-------------------------------------

# # WARNING overcommit_memory is set to 0! Background save may fail under low memory condition. To fix this issue add 'vm.overcommit_memory = 1' to /etc/sysctl.conf and then reboot or run the command 'sysctl vm.overcommit_memory=1' for this to take effect.
# redis_txns_1  | 1:M 12 Mar 2021 14:40:18.085 * Ready to accept connections

# дает docker-compose.yml

# miner_1       | flag provided but not defined: -msk_file
# miner_1       | Usage of ./bin/miner:
# miner_1       |   -delay_file string
# miner_1       |         delay_file
# miner_1       |   -deployment_mode int
# miner_1       |         deployment_mode (default 2)
# miner_1       |   -keys_file string
# miner_1       | flag provided but not defined: -msk_file
# miner_1       |         keys_file
# miner_1       |   -magic_block_file string
# miner_1       |         magic_block_file
# miner_1       | Usage of ./bin/miner:
# miner_1       |   -delay_file string
# miner_1       |         delay_file
# miner_1       |   -deployment_mode int
# miner_1       |         deployment_mode (default 2)
# miner_1       |   -keys_file string
# miner_1       |         keys_file
# miner_1       |   -magic_block_file string
# miner_1       |         magic_block_file

 
