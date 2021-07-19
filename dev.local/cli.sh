#!/bin/bash

root=$(pwd)

cd ../code/go/0chain.net
code=$(pwd)

cd $root


ips=`ifconfig | grep "inet " | grep 198.18.0 | wc -l`

#fix docker network issue for Mac OS X platform
if [ "$(uname)" == "Darwin" ] && [ $ips != 31 ]
then
    # 0dns
    sudo ifconfig lo0 alias 198.18.0.98
    # sharders
    sudo ifconfig lo0 alias 198.18.0.81
    sudo ifconfig lo0 alias 198.18.0.82
    sudo ifconfig lo0 alias 198.18.0.83
    sudo ifconfig lo0 alias 198.18.0.84
    sudo ifconfig lo0 alias 198.18.0.85
    sudo ifconfig lo0 alias 198.18.0.86
    sudo ifconfig lo0 alias 198.18.0.87
    sudo ifconfig lo0 alias 198.18.0.88
    # miners
    sudo ifconfig lo0 alias 198.18.0.71
    sudo ifconfig lo0 alias 198.18.0.72
    sudo ifconfig lo0 alias 198.18.0.73
    sudo ifconfig lo0 alias 198.18.0.74
    sudo ifconfig lo0 alias 198.18.0.75
    sudo ifconfig lo0 alias 198.18.0.76
    sudo ifconfig lo0 alias 198.18.0.77
    sudo ifconfig lo0 alias 198.18.0.78
    # blobbers
    sudo ifconfig lo0 alias 198.18.0.91
    sudo ifconfig lo0 alias 198.18.0.92
    sudo ifconfig lo0 alias 198.18.0.93
    sudo ifconfig lo0 alias 198.18.0.94
    sudo ifconfig lo0 alias 198.18.0.95
    sudo ifconfig lo0 alias 198.18.0.96
    sudo ifconfig lo0 alias 198.18.0.97
    # validators
    sudo ifconfig lo0 alias 198.18.0.61
    sudo ifconfig lo0 alias 198.18.0.62
    sudo ifconfig lo0 alias 198.18.0.63
    sudo ifconfig lo0 alias 198.18.0.64
    sudo ifconfig lo0 alias 198.18.0.65
    sudo ifconfig lo0 alias 198.18.0.66
    sudo ifconfig lo0 alias 198.18.0.67
fi

hostname=`ifconfig | grep "inet " | grep -Fv 127.0.0.1 | grep broadcast | awk '{print $2}'`


cleanAll() {
    cd $root
    rm -rf ./data && echo "data is removed"
}

install_debuggger() {
    [ -d ../code/go/0chain.net/.vscode ] || mkdir -p ../code/go/0chain.net/.vscode
    sed "s/Hostname/$hostname/g" launch.json > ../code/go/0chain.net/.vscode/launch.json
    echo "debugbbers are installed"
}

clean_sharder () {
    echo "clean sharder $i"

    cd $root

    rm -rf "./data/sharder$i"
}

setup_sharder_runtime() {
    echo ""
    echo "Prepare sharder $i: config, files, data, log .."
    cd $root
    [ -d ./data/sharder$i ] && rm -rf ./data/sharder$i

    mkdir -p ./data/sharder$i

    cp -r ../docker.local/config "./data/sharder$i/"

    cd  ./data/sharder$i


    find ./config -name "0chain.yaml" -exec sed -i '' 's/level: "debug"/level: "error"/g' {} \;
    find ./config -name "0chain.yaml" -exec sed -i '' "s/console: false/console: true/g" {} \;
    find ./config -name "0chain.yaml" -exec sed -i '' "s/#    host: cassandra/    host: 127.0.0.1/g" {} \;
    find ./config -name "0chain.yaml" -exec sed -i '' "s/#    port: 9042/    port: 904$i/g" {} \;


    [ -d ./data/blocks ] || mkdir -p ./data/blocks
    [ -d ./data/rocksdb ] || mkdir -p ./data/rocksdb
    [ -d ./log ] || mkdir ./log

    cd $root
}

start_sharder_selector() {
echo "
**********************************************
            Sharder CLI
**********************************************"

    echo " "
    echo "Please select which sharder are you working on: "

     select i in "1" "2";  do
        case $i in
            "1"     )   setup_sharder_runtime; start_sharder_cli;     break;;
            "2"     )   setup_sharder_runtime; start_sharder_cli;     break;;
        esac
    done

}

start_sharder(){

    cd $code

    # Build libzstd with local repo
    # FIXME: Change this after https://github.com/valyala/gozstd/issues/6 is fixed.
    find . -name "go.mod" -exec sed -i '' "/replace github.com\/valyala\/gozstd/d" {} \;
    echo "replace github.com/valyala/gozstd => ../../../../../valyala/gozstd" >> ./go.mod


    cd ./sharder/sharder

    # Build bls with CGO_LDFLAGS and CGO_CPPFLAGS to fix `ld: library not found for -lcrypto`
    export CGO_LDFLAGS="-L/usr/local/opt/openssl@1.1/lib"
    export CGO_CPPFLAGS="-I/usr/local/opt/openssl@1.1/include"

    GIT_COMMIT=$GIT_COMMIT
    go build -o $root/data/sharder$i/sharder -v -tags bn256 -gcflags "all=-N -l" -ldflags "-X 0chain.net/core/build.BuildTag=$GIT_COMMIT"

    cd $root/data/sharder$i/
    ./sharder --deployment_mode 0 --keys_file ./config/b0snode$i_keys.txt --minio_file ./config/minio_config.txt
}


start_sharder_cli() {




echo "
**********************************************
            Sharder $i
**********************************************"

    echo " "
    echo "Please select what are you working on: "

    select f in "install cassandra" "start sharder" "clean sharder"; do
        case $f in
            "install cassandra"     )   cd $root && ./install_cassandra.sh $i;      ;;
            "start sharder"         )   cd $root && start_sharder;                  ;;
            "clean sharder"         )   cd $root && clean_sharder                   ;;
        esac
    done
}


echo "
**********************************************
  Welcome to sharder/miner development CLI 
**********************************************

"

echo "Hostname: $hostname"


echo " "
echo "Please select what are you working on: "

select i in "install [gozstd,rocksdb]" "sharder" "miner" "clean all" "install debugers on .vscode/launch.json"; do
    case $i in
        "install [gozstd,rocksdb]"                  ) ./install_dep.sh          ;;
        "sharder"                                   ) start_sharder_selector    ;;
        "miner"                                     ) start_miner_selector      ;;
        "clean all"                                 ) cleanAll                  ;;
        "install debugers on .vscode/launch.json"   ) install_debuggger         ;;
    esac
done

