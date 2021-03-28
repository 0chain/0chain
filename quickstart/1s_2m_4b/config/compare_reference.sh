#!/bin/bash

. ../base/paths.sh

#---------------------------------------------------

diff -rq ./reference/0Chain/docker.local $zChain_Root/docker.local | grep "^Files.*differ$"

diff -rq ./reference/0dns/docker.local $zDNS_Root/docker.local | grep "^Files.*differ$"

diff -rq ./reference/blobber/docker.local $zBlober_Root/docker.local | grep "^Files.*differ$"
diff -rq ./reference/blobber/config $zBlober_Root/config | grep "^Files.*differ$"



