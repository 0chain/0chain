#!/bin/sh

# (run inside remote 0chain directory)

# clean up without sudo being a member of the docker group

set -x

local_0dns_path="${1}"

if [ -z "${local_0dns_path}" ]; then
    echo "use: sh 0dns-docker-clean.sh ../0dns"
    exit 1
fi

echo "0dns path: ${local_0dns_path}"

abs_0dns_path="$(readlink -f ${local_0dns_path})"

# inside 0chain directory (moving to 0dns directory)
cd "${local_0dns_path}" && \
docker run \
    -v "${abs_0dns_path}/docker.local/0dns/mongodata:/data/db" \
    -v "${abs_0dns_path}/docker.local/config:/0dns/config"     \
    -v "${abs_0dns_path}/docker.local/0dns/log:/0dns/log"      \
    alpine:latest \
    /bin/sh -c 'rm -rf /0dns/log/* && rm -rf /data/db/*'
