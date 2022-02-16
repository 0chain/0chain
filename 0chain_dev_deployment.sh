sharderCount=1
minerCount=3
blobberCount=1

while getopts s:m:b: flag
do
    case "${flag}" in
        s) sharderCount=${OPTARG};;
        m) minerCount=${OPTARG};;
        b) blobberCount=${OPTARG};;
    esac
done

if [ "$(uname)" == "Darwin" ]; then
    # Do something under Mac OS X platform  
    brew install gnu-sed
    alias sed='gsed'
    brew install jq
else
    sudo apt install jq -y
fi
git clone https://github.com/0chain/0chain.git
git clone https://github.com/0chain/blobber.git
git clone https://github.com/0chain/0dns.git

cp 0chain/docker.local/config/b0magicBlock_4_miners_1_sharder.json 0dns/docker.local/config/magic_block.json


cd 0dns

sed -i 's/use_https: true/use_https: false/g' docker.local/config/0dns.yaml
sed -i 's/use_path: true/use_path: false/g' docker.local/config/0dns.yaml
./docker.local/bin/build.sh

cd ../0chain
./docker.local/bin/setup_network.sh

cd ../0dns
./docker.local/bin/start.sh

cd ../0chain
./docker.local/bin/init.setup.sh
sudo chmod -R a+wxr ./docker.local/miner*/*
sudo chmod -R a+wxr ./docker.local/sharder*/*
./docker.local/bin/build.base.sh
./docker.local/bin/build.sharders.sh
./docker.local/bin/build.miners.sh

cd ../blobber
./docker.local/bin/blobber.init.setup.sh
sudo chmod -R a+wxr ./docker.local/blobber*/*
./docker.local/bin/build.base.sh
./docker.local/bin/build.blobber.sh
./docker.local/bin/build.validator.sh

cd ../0chain
cd docker.local
for i in $(seq 1 $sharderCount)
do
   cd sharder$i
   SHARDER=$i docker-compose -p sharder"$i" -f ../build.sharder/b0docker-compose.yml up -d
   cd ../
done
for i in $(seq 1 $minerCount)
do
   cd miner$i
   MINER=$i docker-compose -p miner"$i" -f ../build.miner/b0docker-compose.yml up -d
   cd ../
done

cd ../../
git clone https://github.com/0chain/zboxcli.git

cd zboxcli
make install
mkdir $HOME/.zcn
cp network/one.yaml $HOME/.zcn/config.yaml
sed -i 's|block_worker: https://one.devnet-0chain.net/dns|block_worker: http://198.18.0.98:9091|g' $HOME/.zcn/config.yaml
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

for i in {1..4}
do
echo "Waiting for 30 seconds"
sleep 30
./zbox register | grep 'Wallet registered' &> /dev/null
if [ $? == 0 ]; then # If something is wrong check this condition.
    echo "Wallet registered"
    break;
fi
done
client_id=$(jq ".client_id" $HOME/.zcn/wallet.json -r)

cd ../blobber
sed -i "s/delegate_wallet: '2f34516ed8c567089b7b5572b12950db34a62a07e16770da14b15b170d0d60a9'/delegate_wallet: '$client_id'/g" config/0chain_blobber.yaml
sed -i "s/delegate_wallet: '86b147de5be951a5ab0c6b1732596686185c5e11c3bdf76a112183f828f39ba1'/delegate_wallet: '$client_id'/g" config/0chain_validator.yaml
sudo chmod -R a+wxr ./docker.local/blobber*/*
cd docker.local
for i in $(seq 1 $blobberCount)
do
   cd blobber$i
   BLOBBER=$i docker-compose -p blobber"$i" -f ../b0docker-compose.yml up -d
   cd ../
done

cd ../../
git clone https://github.com/0chain/zwalletcli.git

cd zwalletcli
export PATH=$PATH:/usr/local/go/bin
make install
