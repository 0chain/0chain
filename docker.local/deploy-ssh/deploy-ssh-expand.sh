#/bin/sh

# run inside remote 0chain directory

set -x

##
## 1st and the singe argument must be external IP address of the server
##

# patch localhost -> real IP address
ip_address="${@}"

if [ -z "${ip_address}" ]; then
	echo "use: sh docker.local/bin/deploy-ssh-expand.sh 'ip_address'"
	exit 1
fi

echo "IP address: ${ip_address}"

###
### patch localhost to given IP (script argument)
###

echo "patch localhost to given IP address"

# patch magic block file
temp_mb=$(mktemp)
jq '.miners.nodes[].host = "'"${ip_address}"'"' docker.local/config/b0magicBlock_4_miners_1_sharder.json > "${temp_mb}"
mv -v "${temp_mb}" docker.local/config/b0magicBlock_4_miners_1_sharder.json

# patch *_keys.txt files for non-genesis nodes
for n in $(seq 2 3)
do
	sed -i 's/localhost/'"${ip_address}"'/g' "docker.local/config/b0snode${n}_keys.txt"
done
for n in $(seq 5 8)
do
	sed -i 's/localhost/'"${ip_address}"'/g' "docker.local/config/b0mnode${n}_keys.txt"
done

# patch 0dns_url in 0chain.yaml (yq v3 works another way then v2)
# yq -i -y '.["network"]["0dns_url"] = "'"http://${ip_address}:9091/"'"' docker.local/config/0chain.yaml

if [ "${ip_address}"="localhost" ]
then
	yq w -i docker.local/config/0chain.yaml network.0dns_url "http://0dns_0dns_1:9091/"
else
	yq w -i docker.local/config/0chain.yaml network.0dns_url "http://${ip_address}:9091/"
fi

# setup nodes

echo "setup 0chain directories"
./docker.local/bin/init.setup.sh
echo "setup 0dns directories"
mkdir -pv ../0dns/docker.local/config
mkdir -pv ../0dns/docker.local/bin
mkdir -pv ../0dns/docker.local/0dns/log
cp -v docker.local/deploy-ssh/0dns.yaml ../0dns/docker.local/config/
cp -v docker.local/config/b0magicBlock_4_miners_1_sharder.json \
    ../0dns/docker.local/config/magic_block.json
cp -v docker.local/deploy-ssh/0dns-docker-compose.yml \
    ../0dns/docker.local/
cp -v docker.local/deploy-ssh/0dns.start.sh \
    ../0dns/docker.local/bin
cp -v docker.local/deploy-ssh/0dns.stop.sh \
    ../0dns/docker.local/bin
chmod +x ../0dns/docker.local/bin/0dns.start.sh
chmod +x ../0dns/docker.local/bin/0dns.stop.sh

echo "setup docker network"
./docker.local/bin/setup_network.sh

echo "stop current running, if any"
for i in $(seq 1 8)
do
  sudo systemctl stop "miner${i}" || true
done

for i in $(seq 1 3)
do
  sudo systemctl stop "sharder${i}" || true
done

sudo systemctl stop 0dns || true

echo "cleanup containers and volumes"
./docker.local/bin/docker-clean.sh                      # cleanup 0chain
sh docker.local/deploy-ssh/0dns-docker-clean.sh ../0dns # cleanup 0dns

# systemd services
#

echo "create or update units"

# miners services
#

for i in $(seq 1 8)
do
  cat > miner${i}.service << EOF
[Unit]
After=network.target
After=multi-user.target
Requires=docker.service,0dns.service
Description=0chain/miner${i}

[Service]
Type=simple
WorkingDirectory=$(pwd)/docker.local/miner${i}
User=$(id -nu)
Group=$(id -ng)
ExecStart=$(pwd)/docker.local/bin/start.b0miner.sh
ExecStop=$(pwd)/docker.local/bin/stop.b0miner.sh
TimeoutSec=30
RestartSec=15
Restart=always

[Install]
WantedBy=multi-user.target
EOF
	sudo mv -v miner${i}.service /etc/systemd/system/
done

# sharders services
#

for i in $(seq 1 3)
do
  cat > sharder${i}.service << EOF
[Unit]
After=network.target
After=multi-user.target
Requires=docker.service,0dns.service
Description=0chain/sharder${i}

[Service]
Type=simple
WorkingDirectory=$(pwd)/docker.local/sharder${i}
User=$(id -nu)
Group=$(id -ng)
ExecStart=$(pwd)/docker.local/bin/start.b0sharder.sh
ExecStop=$(pwd)/docker.local/bin/stop.b0sharder.sh
TimeoutSec=180
RestartSec=15
Restart=always

[Install]
WantedBy=multi-user.target
EOF
	sudo mv -v sharder${i}.service /etc/systemd/system/
done

# 0dns service
#

cat > '0dns.service' << EOF
[Unit]
After=network.target
After=multi-user.target
Requires=docker.service
Description=0chain/0dns

[Service]
Type=simple
WorkingDirectory=$(readlink -f ../0dns)/
User=$(id -nu)
Group=$(id -ng)
ExecStart=$(readlink -f ../0dns)/docker.local/bin/0dns.start.sh
ExecStop=$(readlink -f ../0dns)/docker.local/bin/0dns.stop.sh
TimeoutSec=180
RestartSec=15
Restart=always

[Install]
WantedBy=multi-user.target
EOF
sudo mv -v 0dns.service /etc/systemd/system/

echo "reload systemd daemon"
sudo systemctl daemon-reload

echo "done, no units started, start them manually"
