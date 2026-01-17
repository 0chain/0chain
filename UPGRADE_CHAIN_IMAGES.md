# Upgrading Chain Images (Miner & Sharder)

This guide explains how to upgrade miner and sharder images on dev/staging environments.

## Prerequisites

- SSH access to dev servers (dev1.zus.network, dev2.zus.network, dev3.zus.network)
- New Docker image tags available on Docker Hub (0chaindev/miner, 0chaindev/sharder)
- Root access on the servers

## Image Build & Publish

### 1. Trigger GitHub Actions Workflow

Images are built and published via GitHub Actions on PR or push to monitored branches.

**Workflow file:** `.github/workflows/build-&-publish-docker-image.yml`

**Image tags:**
- PR branches: `pr-{PR_NUMBER}-{SHORT_SHA}` (e.g., `pr-3469-624e2575`)
- Master branch: `master-{SHORT_SHA}`
- Staging branch: `staging` (latest)

**Check build status:**
```
https://github.com/0chain/0chain/actions
```

### 2. Local Testing (Optional)

Before deploying, you can test builds locally:

```bash
cd /path/to/0chain

# Build base images
./docker.local/bin/build.base.sh

# Build miner
./docker.local/bin/build.miners.sh

# Build sharder
./docker.local/bin/build.sharders.sh

# Verify images run as expected
docker run --rm miner:latest whoami    # Should show: root
docker run --rm sharder:latest whoami  # Should show: root
```

## Deployment to Dev Servers

### Option A: Using 0helm Workflow (Recommended)

Deploy via the `dev_activeset_cicd.yaml` workflow in the 0helm repository:

1. Go to: `https://github.com/0chain/0helm/actions/workflows/dev_activeset_cicd.yaml`
2. Click "Run workflow"
3. Enter the chain image tag (e.g., `pr-3469-624e25751`)
4. Select target environment
5. The workflow will automatically update and deploy miner/sharder images

### Option B: Manual Deployment

#### Miner Upgrade

```bash
# SSH to dev server (e.g., dev1.zus.network)
ssh root@dev1.zus.network

cd /root/0chain/docker.local/miner1
MID=1

# 1. Update image tag in compose file
IMAGE_TAG="pr-3469-624e2575"  # Replace with your tag
sed -i "s|image: 0chaindev/miner:.*|image: 0chaindev/miner:${IMAGE_TAG}|" \
  ../build.miner/p0docker-compose.yml

# 2. Pull new image
docker pull 0chaindev/miner:${IMAGE_TAG}

# 3. Stop and remove old container
docker rm -f miner${MID} redis${MID} redis_txns${MID} || true

# 4. Start miner
MINER=${MID} docker compose -p miner${MID} \
  -f ../build.miner/p0docker-compose.yml up -d

# 5. Verify
docker logs -f --tail 100 miner${MID}
```

#### Sharder Upgrade

```bash
# SSH to dev server (e.g., dev1.zus.network)
ssh root@dev1.zus.network

cd /root/0chain/docker.local/sharder1

# 1. Update image tag in compose file
IMAGE_TAG="pr-3469-624e2575"  # Replace with your tag
sed -i "s|image: 0chaindev/sharder:.*|image: 0chaindev/sharder:${IMAGE_TAG}|" \
  ../build.sharder/p0docker-compose.yml

# 2. Pull new image
docker pull 0chaindev/sharder:${IMAGE_TAG}

# 3. Stop and remove old containers
docker rm -f sharder${MID} postgres${MID} || true

# 4. Start sharder
SHARDER=${MID} docker compose -p sharder${MID} \
  -f ../build.sharder/p0docker-compose.yml up -d

# 5. Verify
docker logs -f --tail 100 sharder${MID}
```

## Troubleshooting

### Permission Denied Errors

If you see `permission denied` errors for `/0chain/data/*` or `/0chain/log`:

**Cause:** Image runs as non-root user (uid 1000) but mounted directories are owned by root.

**Fix:**
```bash
# For miner
MID=1
sudo chown -R 1000:1000 \
  /root/0chain/docker.local/miner${MID}/data \
  /root/0chain/docker.local/miner${MID}/log

# For sharder
SID=1
sudo chown -R 1000:1000 \
  /mnt/hdd/sharder${SID}/data/blocks \
  /mnt/hdd/sharder${SID}/data/rocksdb \
  /mnt/hdd/sharder${SID}/data/cache \
  /mnt/hdd/sharder${SID}/log

# DO NOT change postgres directory ownership!
```

**Note:** Current images (as of commit 624e25751) run as **root**, so this should not be needed.

### Docker Network IP Address Errors

If you see `invalid IPv4 address: 198.18.2.` error:

**Cause:** `$SHARDER` or `$MINER` environment variable is not set.

**Fix:**
```bash
# Make sure to set the variable before docker compose
SHARDER=1 docker compose -p sharder1 -f ../build.sharder/p0docker-compose.yml up -d
# NOT: docker compose ... (without SHARDER= prefix)
```

### Container Keeps Restarting

**Check logs:**
```bash
docker logs --tail 200 miner1
docker logs --tail 200 sharder1
```

**Common issues:**
1. Permission errors → Fix ownership (see above)
2. Config file errors → Check `/root/0chain/docker.local/config/0chain.yaml`
3. Network issues → Verify `testnet0` network exists: `docker network ls`
4. Database errors (sharder only) → Check postgres logs: `docker logs postgres1`

### Docker Compose Version Issues

If you see `KeyError: 'ContainerConfig'` errors:

**Cause:** Docker Compose v1 compatibility issues.

**Workaround:**
```bash
# Remove container explicitly before starting
docker rm -f miner1 || true
docker compose up -d  # Without --force-recreate

# Or upgrade to Docker Compose v2
```

## Verify Current Image Tags

### Check Local Images
```bash
# List all miner images
docker images 0chaindev/miner

# List all sharder images
docker images 0chaindev/sharder

# Show full image IDs (useful for exact matching)
docker images --no-trunc | grep -E "miner|sharder"
```

### Check Running Container's Image
```bash
# For miner
docker inspect miner1 --format='{{.Config.Image}}'

# For sharder
docker inspect sharder1 --format='{{.Config.Image}}'

# More detailed info including image digest
docker inspect miner1 | jq -r '.[0] | {Image: .Config.Image, ImageID: .Image}'
```

### Check What's Configured in Docker Compose
```bash
# For miner
grep "image:" /root/0chain/docker.local/build.miner/p0docker-compose.yml

# For sharder
grep "image:" /root/0chain/docker.local/build.sharder/p0docker-compose.yml
```

### Check Image on Docker Hub
```bash
# List all tags for miner on Docker Hub
curl -s https://hub.docker.com/v2/repositories/0chaindev/miner/tags/ | jq -r '.results[].name' | head -20

# Check if specific tag exists
TAG="pr-3469-624e25751"
curl -s https://hub.docker.com/v2/repositories/0chaindev/miner/tags/${TAG}/ | jq
```

## Health Checks

After upgrade, verify the nodes are healthy:

### Miner Health Check
```bash
MID=1
curl -s http://localhost:707${MID}/_health_check | jq
```

Expected response:
```json
{
  "ready": true,
  "status": "ready"
}
```

### Sharder Health Check
```bash
SID=1
curl -s http://localhost:717${SID}/_health_check | jq
```

Expected response:
```json
{
  "ready": true,
  "status": "ready"
}
```

## Rollback

If the new image causes issues:

```bash
# 1. Find previous working tag
docker images | grep -E "miner|sharder"

# 2. Update compose file with old tag
OLD_TAG="previous-working-tag"
sed -i "s|image: 0chaindev/miner:.*|image: 0chaindev/miner:${OLD_TAG}|" \
  ../build.miner/p0docker-compose.yml

# 3. Restart with old image
docker rm -f miner1
MINER=1 docker compose -p miner1 -f ../build.miner/p0docker-compose.yml up -d
```

## Multiple Server Deployment

For deploying to all dev servers (dev1, dev2, dev3):

```bash
IMAGE_TAG="pr-3469-624e2575"
SERVERS="dev1.zus.network dev2.zus.network dev3.zus.network"

for SERVER in $SERVERS; do
  echo "Upgrading miner on $SERVER..."
  ssh root@$SERVER "cd /root/0chain/docker.local/miner1 && \
    sed -i 's|image: 0chaindev/miner:.*|image: 0chaindev/miner:${IMAGE_TAG}|' \
      ../build.miner/p0docker-compose.yml && \
    docker pull 0chaindev/miner:${IMAGE_TAG} && \
    docker rm -f miner1 redis1 redis_txns1 || true && \
    MINER=1 docker compose -p miner1 -f ../build.miner/p0docker-compose.yml up -d && \
    sleep 5 && docker logs --tail 50 miner1"
done
```

## Important Notes

1. **Always pull the image** before restarting containers to ensure you get the latest version
2. **Use the correct variable name**: `MINER=` for miners, `SHARDER=` for sharders
3. **Check logs immediately** after starting to catch startup errors
4. **Coordinate upgrades** across servers to minimize downtime
5. **Test in dev** before deploying to staging/production
6. **Keep old images** for quick rollback: `docker images --no-trunc | grep -E "miner|sharder"`

## Configuration Changes

If you need to change node configuration (e.g., cache path, ports):

**Config location:** `/root/0chain/docker.local/config/0chain.yaml`

After changing config:
```bash
# Restart the affected service
docker restart miner1
# OR
docker restart sharder1
```

## Cleanup

Remove old/unused images to save disk space:

```bash
# List images
docker images | grep -E "miner|sharder"

# Remove specific old image
docker rmi 0chaindev/miner:old-tag-name

# Remove all dangling images
docker image prune -f
```

## References

- GitHub Actions Workflow: `.github/workflows/build-&-publish-docker-image.yml`
- Docker Hub: https://hub.docker.com/u/0chaindev
- 0helm Deployment: https://github.com/0chain/0helm
