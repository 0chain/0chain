#!/bin/sh

sed -i "s|ethereum_node_url:\ .*|ethereum_node_url: $1|" config/config.yaml