#!/bin/bash

. ../base/paths.sh

#---------------------------------------------------

echo "0Chain:"

diff -rq ./reference/0Chain/docker.local ./1s_2m_4b/0Chain/docker.local | grep "^Files.*differ$"


echo "0dns:"

diff -rq ./reference/0dns/docker.local ./1s_2m_4b/0dns/docker.local | grep "^Files.*differ$"

echo "blobber/docker.local:"

diff -rq ./reference/blobber/docker.local ./1s_2m_4b/blobber/docker.local | grep "^Files.*differ$"

echo "blobber/config"

diff -rq ./reference/blobber/config ./1s_2m_4b/blobber/config | grep "^Files.*differ$"



