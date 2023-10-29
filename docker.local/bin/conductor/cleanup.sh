#!/bin/bash

cd 0chain;
rm -rf docker.local/miner*/;
rm -rf docker.local/sharder*/;
./docker.local/bin/init.setup.sh;

cd ../blobber;
rm -rf docker.local/blobber*/;
rm -rf docker.local/validator*/;
./docker.local/bin/blobber.init.setup.sh;

cd ../0dns;
rm -rf docker.local/0dns/;
./docker.local/bin/init.sh;

cd ../0box;
rm -rf docker.local/0box/;
./docker.local/bin/init.sh;