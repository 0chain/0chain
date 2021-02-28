#!/bin/sh

redis-cli shutdown nosave
redis-cli -p 6479 shutdown nosave
gnome-terminal -- bash -c "redis-server; exec bash"
gnome-terminal -- bash -c "redis-server --port 6479; exec bash"
