#!/bin/sh

redis-cli shutdown nosave
service redis-server stop
redis-cli -p 6479 shutdown nosave
gnome-terminal -- bash -c \
  "docker run \
      --rm \
      -p 6379:6379 \
      --name redis    \
      -e ALLOW_EMPTY_PASSWORD=yes \
      redis:latest; exec bash"

gnome-terminal -- bash -c \
  "docker run \
      --rm \
      -p 6479:6479 \
      --name redis_txns    \
      -e ALLOW_EMPTY_PASSWORD=yes \
      redis:latest; exec bash"


