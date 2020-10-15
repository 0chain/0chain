#/bin/sh

set -x

# setup nodes

echo "setup directories"
./docker.local/bin/init.setup.sh

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

echo "cleanup directories"
./docker.local/bin/docker-clean.sh

echo "create or update units"
for i in $(seq 1 8)
do
  cat > miner${i}.service << EOF
[Unit]
After=network.target
After=multi-user.target
Requires=docker.service
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

for i in $(seq 1 3)
do
  cat > sharder${i}.service << EOF
[Unit]
After=network.target
After=multi-user.target
Requires=docker.service
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

echo "reload systemd daemon"
sudo systemctl daemon-reload

echo "done, no units started, start them manually"
