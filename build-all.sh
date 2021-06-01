#!/bin/sh

./docker.local/bin/build.base.sh
./docker.local/bin/build.miners.sh
./docker.local/bin/build.sharders.sh
./docker.local/bin/sync_clock.sh