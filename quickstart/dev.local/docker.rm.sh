#!/bin/bash

docker rm --force `docker ps -aq --filter "label=zchain"` && rm -rf ./data