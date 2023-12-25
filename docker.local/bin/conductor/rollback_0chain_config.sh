#!/bin/bash

set -e

echo "rollback 0chain config";

[ -e docker.local/config/0chain.yaml.bak ] && mv docker.local/config/0chain.yaml.bak docker.local/config/0chain.yaml || true