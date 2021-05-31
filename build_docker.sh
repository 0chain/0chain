#!/usr/bin/env bash
ZCM="miner"
ZCS="sharder"
read -p "Provide the docker image tag name: " TAG
read -p "Provide the github organisation name[default:-0chaintest]: " organisation
echo "${organisation:-0chaintest}/${ZCM}:$TAG"
echo "${organisation:-0chaintest}/${ZCS}:$TAG"

REGISTRY_MINER="${organisation:-0chaintest}/${ZCM}"
REGISTRY_SHARDER="${organisation:-0chaintest}/${ZCS}"
DOCKERFILE_MINER="docker.local/build.miner/Dockerfile"
DOCKERFILE_SHARDER="docker.local/build.sharder/Dockerfile"
ZCHAIN_BUILDBASE="zchain_build_base"
ZCHAIN_BUILDRUN="zchain_run_base"
GIT_COMMIT=$(git rev-list -1 HEAD)
echo $GIT_COMMIT
if [ -n "$TAG" ]; then
echo " $TAG is the tage name provided"
echo -e " Creating 0chain docker the base images..\n"
sudo docker build -f docker.local/build.base/Dockerfile.build_base . -t ${ZCHAIN_BUILDBASE}
sudo docker build -f docker.local/build.base/Dockerfile.run_base   docker.local/build.base -t ${ZCHAIN_BUILDRUN}

sudo docker system info | grep -E 'Username' 1>/dev/null
if [[ $? -ne 0 ]]; then
  docker login
fi
 
echo -e "${ZCM}: Docker image build is started.. \n"
sed 's,%COPY%,COPY --from=miner_build $APP_DIR,g' "$DOCKERFILE_MINER.template" > "$DOCKERFILE_MINER"
sudo docker build --build-arg GIT_COMMIT=$GIT_COMMIT -t ${REGISTRY_MINER}:${TAG} -f "$DOCKERFILE_MINER" .
sudo docker pull ${REGISTRY_MINER}:latest
sudo docker tag ${REGISTRY_MINER}:latest ${REGISTRY_MINER}:stable_latest
echo "Re-tagging the remote latest tag to stable_latest"
sudo docker push ${REGISTRY_MINER}:stable_latest
sudo docker tag ${REGISTRY_MINER}:${TAG} ${REGISTRY_MINER}:latest
echo "Pushing the new latest tag to dockerhub"
sudo docker push ${REGISTRY_MINER}:latest
echo "Pushing the new tag to dockerhub tagged as ${REGISTRY_MINER}:${TAG}"
sudo docker push ${REGISTRY_MINER}:${TAG}

echo -e "${ZCS}: Docker image build is started.. \n"
sudo docker build --build-arg GIT_COMMIT=$GIT_COMMIT -t ${REGISTRY_SHARDER}:${TAG} -f "$DOCKERFILE_SHARDER" .
sudo docker pull ${REGISTRY_SHARDER}:latest
sudo docker tag ${REGISTRY_SHARDER}:latest ${REGISTRY_SHARDER}:stable_latest
echo "Re-tagging the remote latest tag to stable_latest"
sudo docker push ${REGISTRY_SHARDER}:stable_latest
sudo docker tag ${REGISTRY_SHARDER}:${TAG} ${REGISTRY_SHARDER}:latest
echo "Pushing the new latest tag to dockerhub"
sudo docker push ${REGISTRY_SHARDER}:latest
echo "Pushing the new tag to dockerhub tagged as ${REGISTRY_SHARDER}:${TAG}"
sudo docker push ${REGISTRY_SHARDER}:${TAG}
fi
