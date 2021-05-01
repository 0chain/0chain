#!/bin/sh

echo "cleaning 6 blobbers..."
for i in $(seq 1 6)
do
  echo "deleting blobber$i logs"
  rm -rf ./blobber$i/log/*
  echo "deleting blobber$i postgresql data"
  rm -rf ./blobber$i/data/postgresql/*
  echo "deleting blobber$i files"
  rm -rf ./blobber$i/data/files/*
done

echo "cleaned up"
