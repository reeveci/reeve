#!/bin/sh

CURRENT_WORK_DIR=$(pwd)
cd $(dirname $0)/..

DOCKER_REGISTRY=${DOCKER_REGISTRY:-docker.io}

docker build -t $DOCKER_REGISTRY/reeveci/reeve --platform=amd64 -f ./reeve-server/docker/Dockerfile .
docker build -t $DOCKER_REGISTRY/reeveci/reeve-worker --platform=amd64 -f ./reeve-worker/docker/Dockerfile .
docker build -t $DOCKER_REGISTRY/reeveci/reeve-runner --platform=amd64 -f ./reeve-runner/docker/Dockerfile .

cd $CURRENT_WORK_DIR
