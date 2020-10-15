#!/bin/sh

if [ -z "$@" ]; then
	echo "use ./docker.local/bin/deploy-ssh-images.sh 'ssh user@host'"
	exit 1
fi

docker save miner | bzip2 | pv | $@ 'docker load'
docker save sharder | bzip2 | pv | $@ 'docker load'
