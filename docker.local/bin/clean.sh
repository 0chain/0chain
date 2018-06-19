#!/bin/sh

for i in $(seq 1 3)
do
  rm docker.local/miner$i/log/*
done

for i in $(seq 1 3)
do
  rm docker.local/sharder$i/log/*
done
