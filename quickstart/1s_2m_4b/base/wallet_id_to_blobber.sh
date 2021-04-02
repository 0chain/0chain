#!/bin/bash

. ./paths.sh

cd $zChain_Root

#--------------------------------------------------------------------------


client_id=`sed -e 's/^.*"client_id":"\([^"]*\)".*$/\1/' ~/.zcn/wallet.json`

echo $client_id
client_id_len=${#client_id}

if [ "$client_id_len" -ne "64" ]; then
   echo 'ERROR: wallet_id_bo_blobber.sh: "$client_id_len -ne 64"';
   exit;
fi   


#Check /0chain_blobber.yaml
echo "Old values"
line_to_replace=`sed -n -e '/^delegate_wallet: /p' "$zBlober_Root"/config/0chain_blobber.yaml`
echo $line_to_replace

if [ -z "$line_to_replace" ]; then
    echo 'ERROR: wallet_id_bo_blobber.sh: cannot find a delegate_wallet in 0chain_blobber.yaml';
fi

#Check /0chain_validator.yaml
line_to_replace=`sed -n -e '/^delegate_wallet: /p' "$zBlober_Root"/config/0chain_validator.yaml`
echo $line_to_replace

if [ -z "$line_to_replace" ]; then
    echo 'ERROR: wallet_id_bo_blobber.sh: cannot find a delegate_wallet in 0chain_validator.yaml';
fi

# Reaplace

new_line="delegate_wallet: '$client_id'"

sed -i '/^delegate_wallet: /c\'"$new_line"''  $zBlober_Root/config/0chain_blobber.yaml
sed -i '/^delegate_wallet: /c\'"$new_line"''  $zBlober_Root/config/0chain_validator.yaml

echo "New values"

line_to_replace=`sed -n -e '/^delegate_wallet: /p' "$zBlober_Root"/config/0chain_blobber.yaml`
echo $line_to_replace

line_to_replace=`sed -n -e '/^delegate_wallet: /p' "$zBlober_Root"/config/0chain_validator.yaml`
echo $line_to_replace

