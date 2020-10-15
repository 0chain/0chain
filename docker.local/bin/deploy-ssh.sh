#!/bin/sh

if [ -z "$@" ]; then
	echo "use ./docker.local/bin/deploy-ssh-images.sh 'ssh user@host'"
	exit 1
fi

tar -czvf "0chain-ssh-$(git rev-parse HEAD).tar.gz" \
    docker.local/bin/init.setup.sh \
    docker.local/bin/setup_network.sh \
    docker.local/build.sharder/b0docker-compose.yml \
    docker.local/build.miner/b0docker-compose.yml \
    bin/ \
    sql/ \
    docker.local/config/cassandra \
    docker.local/config/redis \
    docker.local/config/b0magicBlock_4_miners_1_sharder.json \
    docker.local/config/b0mnode{1,2,3,4,5,6,7,8}_keys.txt \
    docker.local/config/b0snode{1,2,3}_keys.txt \
    docker.local/config/minio_config.txt \
    docker.local/config/0chain.yaml \
    docker.local/config/sc.yaml \
    docker.local/config/b0owner_keys.txt
#  2>/dev/null
