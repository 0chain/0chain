#!/bin/bash

cd ../base

./clear_all.sh

sleep 1


./0dns_clear_restart.sh

sleep 5


./rebuild_sharders.sh

sleep 30

./rebuild_miners.sh

sleep 30


./zwallet_create.sh

sleep 10

./wallet_id_to_blobber.sh

sleep 1

./rebuild_blobbers.sh

sleep 30