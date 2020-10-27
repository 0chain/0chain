#!/bin/sh

# exit on first error
set -x

# check out provided ssh command
ssh_command="${1}"
ip_address="${2}"

if [ -z "${ssh_command}" ]; then
	echo "use: sh docker.local/deploy-ssh/deploy-ssh.sh 'ssh user@host' '3.14.28.109'"
	exit 1
fi

if [ -z "${ip_address}" ]; then
	echo "use: sh docker.local/deploy-ssh/deploy-ssh.sh 'ssh user@host' '3.14.28.109'"
	exit 1
fi

if [ -z "${ip_address}" ]; then
    echo "use: sh docker.local/deploy-ssh/deploy-ssh.sh 'ssh user@host' '3.14.28.109'"
    exit 1
fi

echo "ssh command:     ${ssh_command}"
echo "server address:  ${ip_address}"

# clean up all previous archives
rm -f 0chain-ssh-*.tar.gz

# use commit hash
commit="$(git rev-parse HEAD)"
archive="0chain-ssh-${commit}.tar.gz"

# create minimal archive
tar -czvf "${archive}" \
    ../0chain/docker.local/bin/init.setup.sh \
    ../0chain/docker.local/bin/setup_network.sh \
    ../0chain/docker.local/bin/docker-clean.sh \
    ../0chain/docker.local/bin/start.b0sharder.sh \
    ../0chain/docker.local/bin/start.b0miner.sh \
    ../0chain/docker.local/bin/stop.b0sharder.sh \
    ../0chain/docker.local/bin/stop.b0miner.sh \
    ../0chain/docker.local/bin/dump_logs.sh \
    ../0chain/docker.local/build.sharder/b0docker-compose.yml \
    ../0chain/docker.local/build.miner/b0docker-compose.yml \
    ../0chain/docker.local/docker-clean/ \
    ../0chain/bin/ \
    ../0chain/sql/ \
    ../0chain/config/cassandra \
    ../0chain/config/redis \
    ../0chain/docker.local/config/cassandra \
    ../0chain/docker.local/config/redis \
    ../0chain/docker.local/config/b0magicBlock_4_miners_1_sharder.json \
    ../0chain/docker.local/config/b0mnode{1,2,3,4,5,6,7,8}_keys.txt \
    ../0chain/docker.local/config/b0snode{1,2,3}_keys.txt \
    ../0chain/docker.local/config/minio_config.txt \
    ../0chain/docker.local/config/0chain.yaml \
    ../0chain/docker.local/config/sc.yaml \
    ../0chain/docker.local/config/b0owner_keys.txt \
    ../0chain/docker.local/deploy-ssh/

# upload the archive
cat "${archive}" | pv | 
    ${ssh_command} 'tar -C ./ -zxvf - && cd 0chain && pwd && sh docker.local/deploy-ssh/deploy-ssh-expand.sh '"${ip_address}"''

# clean the created archive
rm -f 0chain-ssh-*.tar.gz
