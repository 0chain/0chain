#!/bin/sh

RepoRoot=$(git rev-parse --show-toplevel)
rm -r $RepoRoot/local/miner/data
rm -r $RepoRoot/local/miner/log
rm -r $RepoRoot/local/sharder/data
rm -r $RepoRoot/local/sharder/log