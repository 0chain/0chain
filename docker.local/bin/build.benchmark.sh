#!/bin/sh

cmd="build"

docker $cmd -f docker.local/build.benchmarks/Dockerfile . -t zchain_benchmarks

if [ "$#" -ne 3 ]; 
    then echo "illegal number of parameters usage: docker.local/bin/gen_keys.sh signatureScheme [bls0chain or ed25519] absolute_path_to_keyfiles_folder key_file_name"
    exit 
fi

docker run -v "$2":/mykeys -it zchain_genkeys go run encryption/keys/main.go   --signature_scheme "$1" --keys_file_name "$3" --keys_file_path "/mykeys" --generate_keys true  --timestamp true

retVal=$?
if [ $retVal -ne 0 ]; then
    exit $retVal
fi
echo "generated file: $2/$3"
