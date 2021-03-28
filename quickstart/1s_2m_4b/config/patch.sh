#!/bin/bash

. ../base/paths.sh

#---------------------------------------------------

rsync -r ./1s_2m_4b/0Chain/docker.local/ $zChain_Root/docker.local/

rsync -r ./1s_2m_4b/0dns/docker.local/ $zDNS_Root/docker.local/

rsync -r ./1s_2m_4b/blobber/docker.local/ $zBlober_Root/docker.local/

cp ./1s_2m_4b/blobber/config/0chain_blobber.yaml $zBlober_Root/config/0chain_blobber.yaml
cp ./1s_2m_4b/blobber/config/0chain_validator.yaml $zBlober_Root/config/0chain_validator.yaml

#---------------------------------------------------

cp ./1s_2m_4b/zcn/network.yaml ~/.zcn/network.yaml
cp ./1s_2m_4b/zcn/config.yaml ~/.zcn/config.yaml


