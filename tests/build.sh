#!/bin/sh

CURRENT_WORK_DIR=$(pwd)
cd $(dirname $0)/..

docker build -t reeve -f ./reeve-server/docker/Dockerfile .
docker build -t reeve-worker -f ./reeve-worker/docker/Dockerfile .
docker build -t reeve-runner -f ./reeve-runner/docker/Dockerfile .

cd $CURRENT_WORK_DIR

$(dirname $0)/rebuild-plugin.sh
$(dirname $0)/recreate-cert.sh
