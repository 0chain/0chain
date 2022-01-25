#!/bin/bash

#clean all data
cleanAll() {
    cd $root
    rm -rf ./data && echo "config/data are removed"
}

#install vscode debugger
install_debuggger() {
    [ -d ../code/go/0chain.net/.vscode ] || mkdir -p ../code/go/0chain.net/.vscode
    sed "s/Hostname/$hostname/g" launch.json > ../code/go/0chain.net/.vscode/launch.json
    echo "debugbbers are installed"
}