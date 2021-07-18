#!/bin/bash

root=$(pwd)


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
    [ -d ../.vscode ] || mkdir -p ../.vscode
    sed "s/Hostname/$hostname/g" launch.json > ../.vscode/launch.json
    echo "debugbbers are installed"
}

clean_sharder () {
    echo "clean sharder $i"

    cd $root

    rm -rf "./data/sharder$i"
}

start_sharder_selector() {
echo "
**********************************************
            Sharder CLI
**********************************************"

    echo " "
    echo "Please select which shareder are you working on: "

     select i in "1" "2";  do
        case $i in
            "1"     )   start_sharder_cli;     break;;
            "2"     )   start_sharder_cli;     break;;
        esac
    done

}

start_sharder_cli() {
echo "
**********************************************
            Sharder CLI
**********************************************"

    echo " "
    echo "Please select what are you working on: "

    select f in "install cassandra" "start sharder" "clean sharder"; do
        case $f in
            "install cassandra"     )   ./install_cassandra.sh $i;           ;;
            "start sharder"         )   start_blobber;                  break;;
            "clean sharder"         )   clean_sharder                        ;;
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

