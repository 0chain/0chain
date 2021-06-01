#!/bin/sh

docker stop $(docker ps -a -q)
./docker.local/bin/clean.sh
./docker.local/bin/init.setup.sh
./docker.local/bin/sync_clock.sh