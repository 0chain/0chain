#!/bin/sh

RepoRoot=$(git rev-parse --show-toplevel)
mkdir -p $RepoRoot/local/miner/data/rocksdb
mkdir -p $RepoRoot/local/miner/log
mkdir -p $RepoRoot/local/miner/config
cp $RepoRoot/docker.local/config/sc.yaml $RepoRoot/local/miner/config/sc.yaml
cp $RepoRoot/docker.local/config/n2n_delay.yaml $RepoRoot/local/miner/config/n2n_delay.yaml
cp $RepoRoot/docker.local/config/b0owner_keys.txt $RepoRoot/local/miner/config/b0owner_keys.txt
cp $RepoRoot/docker.local/config/initial_state.yaml $RepoRoot/local/miner/config/initial_state.yaml

mkdir -p $RepoRoot/local/sharder/data/blocks
mkdir -p $RepoRoot/local/sharder/data/rocksdb
mkdir -p $RepoRoot/local/sharder/log
mkdir -p $RepoRoot/local/sharder/config
cp $RepoRoot/docker.local/config/sc.yaml $RepoRoot/local/sharder/config/sc.yaml
cp $RepoRoot/docker.local/config/initial_state.yaml $RepoRoot/local/miner/config/initial_state.yaml