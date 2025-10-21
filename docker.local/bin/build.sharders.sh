#!/bin/bash
set -e

# Read configuration from blockchain.config
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ZUS_SETUP_DIR="$(dirname "$(dirname "$SCRIPT_DIR")")"

if [ -f "$ZUS_SETUP_DIR/blockchain.config" ]; then
    source "$ZUS_SETUP_DIR/blockchain.config"
else
    # Default configuration
    NUM_SHARDERS=2
fi

GIT_COMMIT=$(git rev-list -1 HEAD)
echo "$GIT_COMMIT"
echo "Building $NUM_SHARDERS sharders..."

cmd="build"

# generate swagger
#echo "generating swagger.yaml file"
#docker.local/bin/test.swagger.sh

docker $cmd --build-arg GIT_COMMIT="$GIT_COMMIT" -f docker.local/build.sharder/Dockerfile . -t sharder

for i in $(seq 1 $NUM_SHARDERS);
do
  echo "Building sharder $i..."
  SHARDER=$i docker-compose -p sharder$i -f docker.local/build.sharder/docker-compose.yml build --force-rm
done

docker.local/bin/sync_clock.sh
