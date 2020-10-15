#!/bin/sh

ssh_command="${@}"

if [ -z "${ssh_command}" ]; then
	echo "use ./docker.local/bin/deploy-ssh-images.sh 'ssh user@host'"
	exit 1
fi

echo "ssh command: ${ssh_command}"

docker save miner | bzip2 | pv | ${ssh_command} 'docker load'
docker save sharder | bzip2 | pv | ${ssh_command} 'docker load'
