#!/bin/sh

. ./paths.sh

cd $zChain_Root

#----------------------------------------------

docker stop $(docker ps -a -q)

docker rm $(docker ps -a -q)

docker volume prune

sudo rm -rf docker.local/miner*/*

sudo rm -rf docker.local/sharder*/*

sudo docker.local/bin/init.setup.sh

sudo docker.local/bin/build.base.sh


#----------------------------------------------

cd $zBlober_Root

for i in $(seq 1 6)
do

  sudo rm -rf docker.local/blobber$i/*
  mkdir -p docker.local/blobber$i/files
  mkdir -p docker.local/blobber$i/data/postgresql
  mkdir -p docker.local/blobber$i/log	

done


cd $zDNS_Root

sudo rm -rf docker.local/0dns/log/*
sudo rm -rf docker.local/0dns/mongodata/*
mkdir -p docker.local/0dns/log


sudo rm -rf ~/.zcn/allocation.txt
sudo rm -rf ~/.zcn/wallet.json