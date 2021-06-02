#!/bin/bash

. ../base/paths.sh

#---------------------------------------------------

echo "0Chain:"

diff -rq ./reference/0Chain/docker.local $zChain_Root/docker.local


echo "0dns:"

diff -rq ./reference/0dns/docker.local $zDNS_Root/docker.local

echo "blobber/docker.local:"

diff -rq ./reference/blobber/docker.local $zBlober_Root/docker.local

echo "blobber/config"

diff -rq ./reference/blobber/config $zBlober_Root/config



