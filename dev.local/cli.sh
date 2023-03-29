#!/bin/bash

set -e

root=$(pwd)

cd ../code/go/0chain.net
code=$(pwd)
cd $root

#fix docker network issue on MacOS
./cli.ifconfig.sh

hostname=`ifconfig | grep "inet " | grep -Fv 127.0.0.1 | grep broadcast | awk '{print $2}'`


# fixed LIBRARY_PATH
snappy=$(brew --prefix snappy)
lz4=$(brew --prefix lz4)
gmp=$(brew --prefix gmp)
openssl=$(brew --prefix openssl@1.1)
export GMP_DIR=${gmp}
export LIBRARY_PATH="/usr/local/lib:${openssl}/lib:${snappy}/lib:${lz4}/lib:${gmp}/lib"
export LD_LIBRARY_PATH="/usr/local/lib:${openssl}/lib:${snappy}/lib:${lz4}/lib:${gmp}/lib"
export DYLD_LIBRARY_PATH="/usr/local/lib:${openssl}/lib:${snappy}/lib:${lz4}/lib:${gmp}/lib"
export CGO_LDFLAGS="-L/usr/local/lib -lrocksdb -lstdc++ -lm -lz -lbz2 -lsnappy -llz4 -lgmp"
export CGO_CFLAGS="-I/usr/local/include"
export CGO_CPPFLAGS="-I/usr/local/include"
export LDFLAGS="-L/usr/local/lib -lrocksdb -lstdc++ -lm -lz -lbz2 -lsnappy -llz4 -lgmp"
export CFLAGS="-I/usr/local/include"
export CPPFLAGS="-I/usr/local/include"

#include base 
. ./cli.base.sh

#include sharder
. ./cli.sharder.sh

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


#sharder cli
start_sharder_cli() {

echo "
**********************************************
            Sharder $i
**********************************************"

    echo " "
    echo "Please select what are you working on: "

    select f in "install cassandra" "start sharder" "clean sharder"; do
        case $f in
            "install cassandra"     )   cd $root && ./cli.sharder.cassandra.sh $i;      ;;
            "start sharder"         )   cd $root && start_sharder;                  ;;
            "clean sharder"         )   cd $root && clean_sharder                   ;;
        esac
    done
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


#sharder cli
start_sharder_cli() {

echo "
**********************************************
            Sharder $i
**********************************************"

    echo " "
    echo "Please select what are you working on: "

    select f in "install [postgres,cassandra]" "start sharder" "clean sharder"; do
        case $f in
            "install [postgres,cassandra]"     )   cd $root && ./cli.sharder.db.sh $i;           ;;
            "start sharder"                    )   cd $root && start_sharder;                    ;;
            "clean sharder"                    )   cd $root && clean_sharder                     ;;
        esac
    done
}


#include miner
. ./cli.miner.sh

#miner cli
start_miner_cli() {

echo "
**********************************************
            Miner $i
**********************************************"

    echo " "
    echo "Please select what are you working on: "

    select f in "install redis" "start miner" "clean miner"; do
        case $f in
            "install redis"  )   cd $root && ./cli.miner.redis.sh $i;      ;;
            "start miner"    )   cd $root && start_miner;                  ;;
            "clean miner"    )   cd $root && clean_miner                   ;;
        esac
    done
}


start_miner_selector() {
echo "
**********************************************
            Miner CLI
**********************************************"

    echo " "
    echo "Please select which miner are you working on: "

     select i in "1" "2" "3" "4";  do
        case $i in
            "1"     )   setup_miner_runtime; start_miner_cli;     break;;
            "2"     )   setup_miner_runtime; start_miner_cli;     break;;
            "3"     )   setup_miner_runtime; start_miner_cli;     break;;
            "4"     )   setup_miner_runtime; start_miner_cli;     break;;
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

select i in "install [rocksdb,herumi,openssl]" "sharder" "miner" "clean all" "install debugers on .vscode/launch.json"; do
    case $i in
        "install [rocksdb,herumi,openssl]"          ) ./install.deps.sh         ;;
        "sharder"                                   ) start_sharder_selector    ;;
        "miner"                                     ) start_miner_selector      ;;
        "clean all"                                 ) cleanAll                  ;;
        "install debugers on .vscode/launch.json"   ) install_debuggger         ;;
    esac
done
