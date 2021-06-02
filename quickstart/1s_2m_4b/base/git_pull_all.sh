 
#!/bin/bash
set -e

. ./paths.sh


#-------------------------------------------------


if [ ! -d $zChain_Root ]; then
    mkdir -p $zChain_Root
    cd $zChain_Root
    git clone git@github.com:0chain/0chain.git .
    git reset --hard ac6c2253
fi



if [ ! -d $zDNS_Root ]; then
    mkdir -p $zDNS_Root
    cd $zDNS_Root
    git clone git@github.com:0chain/0dns.git .
    git reset --hard d6e1dbb3
fi



if [ ! -d $zBlober_Root ]; then
    mkdir -p $zBlober_Root
    cd $zBlober_Root
    git clone git@github.com:0chain/blobber.git .
    git reset --hard 28c31930
fi    



if [ ! -d $gosdk ]; then
    mkdir -p $gosdk
    cd $gosdk
    git clone git@github.com:0chain/gosdk.git .
    git reset --hard 54902b25
fi




if [ ! -d $zCLI_Root ]; then
    mkdir -p $zCLI_Root
    cd $zCLI_Root
    git clone git@github.com:0chain/zboxcli.git .
    git reset --hard 25f8af1a
fi



if [ ! -d $zWallet_Root ]; then
    mkdir -p $zWallet_Root
    cd $zWallet_Root
    git clone git@github.com:0chain/zwalletcli.git .
    git reset --hard 815813a2
fi