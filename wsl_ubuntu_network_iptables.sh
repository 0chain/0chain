#!/usr/bin/env bash

NETWORK="198.18.0.0/15"

if [ "$(command -v iptables)" ]; then
    sudo iptables -t nat -I OUTPUT --dst "$NETWORK" -p tcp -j REDIRECT
    echo "$(tput setaf 2)""[SUCCESS]""$(tput sgr0)" "NAT entries updated successfully for network $NETWORK. Run iptables -t nat -L OUTPUT to make sure it exists"
    exit 0
fi

echo "$(tput setaf 1)""[FAILURE]""$(tput sgr0)" "\"iptables\" should be installed first to run this script."
exit 1