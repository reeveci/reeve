#!/bin/sh
set -e

if [ -z "$VERSION" ]; then
    echo "Missing VERSION variable"
    exit 1
fi

if ! git show-ref --tags "v$VERSION" --quiet; then
    echo "Missing v$VERSION tag"
    exit 1
fi

DOCKER_REGISTRY=${DOCKER_REGISTRY:-docker.io}

docker tag $DOCKER_REGISTRY/reeveci/reeve $DOCKER_REGISTRY/reeveci/reeve:$VERSION
docker tag $DOCKER_REGISTRY/reeveci/reeve-worker $DOCKER_REGISTRY/reeveci/reeve-worker:$VERSION
docker tag $DOCKER_REGISTRY/reeveci/reeve-runner $DOCKER_REGISTRY/reeveci/reeve-runner:$VERSION

docker push $DOCKER_REGISTRY/reeveci/reeve
docker push $DOCKER_REGISTRY/reeveci/reeve:$VERSION

docker push $DOCKER_REGISTRY/reeveci/reeve-worker
docker push $DOCKER_REGISTRY/reeveci/reeve-worker:$VERSION

docker push $DOCKER_REGISTRY/reeveci/reeve-runner
docker push $DOCKER_REGISTRY/reeveci/reeve-runner:$VERSION
