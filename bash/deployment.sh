#!/bin/bash

sharderCount=2
minerCount=3
blobberCount=6

#each step output
set -x

#Deployment option
echo "Choose deployment option:"
echo "1. Local"
echo "2. Server"
read -p "Enter your choice (1/2): " choice

# directory creation
mkdir zus-repos

cd zus-repos


# 2d-array to pull all required images
declare -A arr
arr[0,0]="pull 0chaindev/sharder:staging"
arr[0,1]="tag 0chaindev/sharder:staging sharder"
arr[0,2]="image rm 0chaindev/sharder:staging"
arr[1,0]="pull 0chaindev/miner:staging"
arr[1,1]="tag 0chaindev/miner:staging miner"
arr[1,2]="image rm pull 0chaindev/miner:staging"
arr[2,0]="pull 0chaindev/blobber:staging"
arr[2,1]="tag 0chaindev/blobber:staging blobber"
arr[2,2]="image rm 0chaindev/blobber:staging"
arr[3,1]="pull 0chaindev/validator:staging"
arr[3,2]="tag 0chaindev/validator:staging validator"
arr[3,3]="image rm 0chaindev/validator:staging"

for ((j=0;j<=3;j++)) do
    #echo  $j
    for ((i=0;i<=3;i++)) do
       docker ${arr[$i,$j]}
    done
    echo
done




# docker pull 0chaindev/sharder:staging 
# docker tag  0chaindev/sharder:staging sharder
# docker pull 0chaindev/miner:staging 
# docker tag 0chaindev/miner:staging miner
# docker pull 0chaindev/validator:staging
# docker tag 0chaindev/validator:staging validator
# docker pull 0chaindev/blobber:staging
# docker tag 0chaindev/blobber:staging blobber



# array to clone required repos
declare -a repo=(

[0]=git@github.com:0chain/0chain.git
[1]=git@github.com:0chain/blobber.git
[2]=git@github.com:0chain/0dns.git

)

for i in ${!repo[@]}; do
  git clone  ${repo[$i]}
done

cp 0chain/docker.local/config/b0magicBlock_4_miners_1_sharder.json 0dns/docker.local/config/magic_block.json

cd 0dns
# Dns chnage for local deployment
if [ "$choice" == "1" ]; then
   CONFIG_FILE="docker.local/config/0dns.yaml"
   sed -i "s/use_https.*/use_https: false/" "$CONFIG_FILE"
   sed -i "s/use_path.*/use_path: false/" "$CONFIG_FILE"
fi
echo " docker build"
docker.local/bin/build.sh

cd ../0chain

docker.local/bin/setup.network.sh

cd ../0dns
docker.local/bin/start.sh

cd ../0chain

docker.local/bin/init.setup.sh

sudo chmod -R a+wxr docker.local/miner*/*
sudo chmod -R a+wxr docker.local/sharder*/*

ls docker.local/

#install mockery
echo "" | /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
eval "$(/home/linuxbrew/.linuxbrew/bin/brew shellenv)"
brew doctor
brew install mockery

# make build mocks
make build-mocks 

# build sharder
#docker pull 0chaindev/sharder:staging 


# build miner
#docker pull 0chaindev/miner:staging 

cd ../blobber

chmod +x docker.local/bin/blobber.init.setup.sh
docker.local/bin/blobber.init.setup.sh
sudo chmod -R a+wxr docker.local/blobber*/*


ls docker.local/
# Dns chnage for local deployment
cd config
if [ "$choice" == "1" ]; then
   CONFIG_FILE="0chain_blobber.yaml"
   
   sed -i 's/block_worker.*/block_worker: http:\/\/198.18.0.98:9091\//' "$CONFIG_FILE"
   
   CONFIG_FILE="0chain_validator.yaml"
   # # sed -i "s/block_worker.*/block_worker: http://198.18.0.98:9091/dns/" "$CONFIG_FILE"
   sed -i 's/block_worker.*/block_worker: http:\/\/198.18.0.98:9091\//' "$CONFIG_FILE"
fi
cd ..

# docker pull 0chaindev/blobber:staging
# docker pull 0chaindev/validator:staging

cd ../0chain/docker.local
#Starting sharder

for i in {1..2}; 
do
   (
     cd sharder$i/
    ../bin/start.b0sharder.sh
    cd ..
   ) &
   
done


echo " completed first loop"
+
# Starting miners 

for i in {1..3}; 
do 
    (
      cd miner$i/
      ../bin/start.b0miner.sh
      cd ..
    ) &
   
done

# # cd ../../blobber/docker.local

# # for i in {1..6}; 
# # do 
# #     (
# #       cd blobber$i/
# #       ../bin/blobber.start_bls.sh
# #       cd ..
# #     ) &
   
# # done

cd ../..

sudo apt update

sudo apt-get install build-essential

git clone https://github.com/0chain/zwalletcli.git

cd zwalletcli

make install

./zwallet

if [ -d "$HOME/.zcn" ]; then

   rm -r $HOME/.zcn

fi



mkdir $HOME/.zcn


cp network/config.yaml $HOME/.zcn/config.yaml
CONFIG_FILE="$HOME/.zcn/config.yaml"
if [ "$choice" == "1" ]; then
  
 sed -i 's/block_worker.*/block_worker: http:\/\/198.18.0.98:9091\//' "$CONFIG_FILE"

fi
echo "miners:" > $HOME/.zcn/network.yaml
for i in $(seq 1 $minerCount)
do
    printf "  - http://localhost:%d\n" $(($i+7070)) >> $HOME/.zcn/network.yaml
done
echo "sharders:" >> $HOME/.zcn/network.yaml
for i in $(seq 1 $sharderCount)
do
    printf "  - http://localhost:%d\n" $(($i+7170)) >> $HOME/.zcn/network.yaml
done

./zwallet create-wallet 

for i in {1..3}; 
do
   (
      ./zwallet faucet --methodName pour --input "new wallet" 
   ) &
   
done

./zwallet getbalance 

client_id="$(grep -Po '"client_id": *\K"[^"]*"' $HOME/.zcn/wallet.json)"
cd ../blobber/config
CONFIG_FILE="0chain_blobber.yaml"
sed -i "s/delegate_wallet.*/delegate_wallet: $client_id/" "$CONFIG_FILE"
CONFIG_FILE="0chain_validator.yaml"
sed -i "s/delegate_wallet.*/delegate_wallet: $client_id/" "$CONFIG_FILE"
cd ../..
cd blobber/docker.local
for i in $(seq 1 $blobberCount)
do
   cd blobber$i
   BLOBBER=$i docker-compose -p blobber"$i" -f ../b0docker-compose.yml up -d
   cd ../
done