#!/bin/sh
set -e

docker run \
  --rm -i \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -e DOCKER_LOGIN_REGISTRY \
  -e DOCKER_LOGIN_USER \
  -e DOCKER_LOGIN_PASSWORD \
  -e REEVE_API_PORT \
  -e REEVE_RUNTIME_ENV \
  -e REEVE_DOCKER_COMMAND \
  -e REEVE_NO_DESCRIPTION \
  --name reeve-runner-$(cat /proc/sys/kernel/random/uuid) \
  $REEVE_RUNNER_IMAGE
