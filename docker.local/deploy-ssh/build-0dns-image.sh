#!/bin/sh

set -x

local_0dns_path="${1}"

if [ -z "${local_0dns_path}" ]; then
	echo "use: sh build-0dns-image.sh ../0dns"
	exit 1
fi

cd "${local_0dns_path}" && docker build -f docker.local/Dockerfile . -t 0dns
