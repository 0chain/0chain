#!/usr/bin/env bash

HMDIR="$HOME/0chain"
ZCHAINDIR="$HOME/0chain/0chain"
ZDNSDIR="$HOME/0chain/0dns"
BLOBBERDIR="$HOME/0chain/blobber"
SHARDER_DIR_NAME="sharder"
MINER_DIR_NAME="miner"
BLOBBER_DIR_NAME="blobber"

while getopts 'cn:' OPTIONS; do
   case "${OPTIONS}" in
    c)
        CLEAN=1
    ;;
    n)
        N=${OPTARG}
    ;;
    *)
        echo "Unsupported flag"
    ;;
   esac
done
shift "$((OPTIND -1))"

case $* in
    "build-sharders")
        cd "$ZCHAINDIR" || exit
        "./docker.local/bin/build.sharders.sh"
    ;;
    "build-miners")
        cd "$ZCHAINDIR" || exit
        "./docker.local/bin/build.miners.sh"
    ;;
    "build-blobber")
        cd "$BLOBBERDIR" || exit
        "./docker.local/bin/build.base.sh"
        "./docker.local/bin/build.blobber.sh"
        "./docker.local/bin/build.validator.sh"
    ;;
    "build-dns")
        cd "$ZDNSDIR" || exit
        "./docker.local/bin/build.sh"
    ;;
    "clean-0chain")
        cd "$ZCHAINDIR" || exit
        sudo sh -c "./docker.local/bin/clean.sh"
    ;;
    "clean-blobber")
        cd "$BLOBBERDIR" || exit
        sudo sh -c "./docker.local/bin/clean.sh"
    ;;
    "clean-dns")
        cd "$ZDNSDIR" || exit
        sudo sh -c "./docker.local/bin/clean.sh"
    ;;
    "run-sharders")
        if (( CLEAN == 1 )); then
            echo "Cleaning ..."
            cd "$ZCHAINDIR" || exit
            sudo rm -rf "$ZCHAINDIR/docker.local/$SHARDER_DIR_NAME*"
            "./docker.local/bin/init.setup.sh"
        fi

        # TODO:: Wait 10 sec if `docker-compose list` still has sharders. Repeat 10 times then die        
        for((i=1;i<=N;i++)); do
            echo "Starting Sharder $i"
            cd "$ZCHAINDIR/docker.local/$SHARDER_DIR_NAME$i" || exit
            "../bin/start.b0sharder.sh" &>/dev/null &
        done
    ;;
    "stop-sharders")
        for((i=1;i<=N;i++)); do
            echo "Stoping Sharder $i"
            cd "$ZCHAINDIR/docker.local/$SHARDER_DIR_NAME$i" || exit
            "../bin/stop.b0sharder.sh" &>/dev/null &
        done
    ;;
    "run-miners")
        if (( CLEAN == 1 )); then
            echo "Cleaning ..."
            cd "$ZCHAINDIR" || exit
            sudo rm -rf "$ZCHAINDIR/docker.local/$MINER_DIR_NAME*"
            "./docker.local/bin/init.setup.sh"
        fi

        # TODO:: Wait 10 sec if `docker-compose list` still has miners. Repeat 10 times then die
        for((i=1;i<=N;i++)); do
            echo "Starting Miner $i"
            cd "$ZCHAINDIR/docker.local/$MINER_DIR_NAME$i" || exit
            "../bin/start.b0miner.sh" &>/dev/null &
        done
    ;;
    "run-dns")
        cd "$ZDNSDIR" || exit
        "./docker.local/bin/start.sh" &>/dev/null &
    ;;
    "stop-miners")
        for((i=1;i<=N;i++)); do
            echo "Stoping Miner $i"
            cd "$ZCHAINDIR/docker.local/$MINER_DIR_NAME$i" || exit
            "../bin/stop.b0miner.sh" &>/dev/null &
        done
    ;;
    "run-blobbers")
        if (( CLEAN == 1 )); then
            echo "Cleaning ..."
            cd "$BLOBBERDIR" || exit
            sudo sh -c "./docker.local/bin/clean.sh"
        fi
        # TODO:: Wait 10 sec if `docker-compose list` still has blobbers. Repeat 10 times then die
        for((i=1;i<=N;i++)); do
            echo "Starting Blobber $i"
            cd "$BLOBBERDIR/docker.local/$BLOBBER_DIR_NAME$i" || exit
            "../bin/blobber.start_bls.sh" &>/dev/null &
        done
    ;;
    "stop-blobbers")
        for((i=1;i<=N;i++)); do
            echo "Stopping Blobber $i"
            cd "$BLOBBERDIR/docker.local/$BLOBBER_DIR_NAME$i" || exit
            "../bin/blobber.stop_bls.sh" &>/dev/null &
        done
    ;;
    *)
        echo "Command not found"
        exit 1
    ;;
esac

cd "$HMDIR" || exit
