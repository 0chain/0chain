#!/bin/sh

set -e

# it's inside 0dns/ minimal remote directory
docker-compose -p 0dns -f docker.local/0dns-docker-compose.yml up
