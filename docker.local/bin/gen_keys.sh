#!/bin/sh

cmd="build"

for arg in "$@"
do
    case $arg in
        -m1|--m1|m1)
        echo "The build will be performed for Apple M1 chip"
        cmd="buildx build --platform linux/amd64"
        shift
        ;;
    esac
done

docker $cmd -f docker.local/build.genkeys/Dockerfile . -t zchain_genkeys

if [ "$#" -ne 3 ];
    then echo "illegal number of parameters usage: docker.local/bin/gen_keys.sh signatureScheme [bls0chain or ed25519] absolute_path_to_keyfiles_folder key_file_name"
    exit
fi

docker run -v "$2":/mykeys -it zchain_genkeys go run -tags bn256 encryption/keys/main.go   --signature_scheme "$1" --keys_file_name "$3" --keys_file_path "/mykeys" --generate_keys true  --timestamp true

retVal=$?
if [ $retVal -ne 0 ]; then
    exit $retVal
fi
echo "generated file: $2/$3"
