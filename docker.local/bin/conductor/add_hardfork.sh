#!/bin/bash

set -e

names=$1
rounds=$2

./zwalletcli/zwallet add-hardfork --names=${names} --rounds=${rounds} 