#!/bin/sh

set -x

ssh_command="${1}"

if [ -z "${ssh_command}" ]; then
	echo "use: sh remote-setup-ubuntu.sh 'ssh user@host'"
	exit 1
fi

echo "ssh command: ${ssh_command}"

${ssh_command} '

set -x

sudo gpasswd -a $USER docker
'

# the relogin required to be added to new group -- docker, the 'newgrp'
# command can't be used in a script

${ssh_command} '

set -x

sudo systemctl start docker
sudo apt-get update && sudo apt-get install docker-compose jq

cd "${HOME}"
wget https://github.com/docker/docker-credential-helpers/releases/download/v0.6.3/docker-credential-secretservice-v0.6.3-amd64.tar.gz
tar -xf docker-credential-secretservice-v0.6.3-amd64.tar.gz
chmod +x docker-credential-secretservice
sudo mv /usr/bin/docker-credential-secretservice /usr/bin/docker-credential-secretservice.bkp || true
sudo mv -v docker-credential-secretservice /usr/bin/
'
