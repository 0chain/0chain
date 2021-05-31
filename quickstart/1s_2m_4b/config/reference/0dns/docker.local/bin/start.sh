#!/bin/sh
PWD=`pwd`

echo Starting 0dns ...

docker-compose -p 0dns -f ../docker-compose.yml up -d
