#!/bin/sh
set -e

docker run \
  --rm -i \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -e DOCKER_LOGIN_REGISTRIES \
  -e DOCKER_LOGIN_REGISTRY \
  -e DOCKER_LOGIN_USER \
  -e DOCKER_LOGIN_PASSWORD \
  -e REEVE_API_PORT \
  -e REEVE_RUNTIME_ENV \
  -e REEVE_DOCKER_COMMAND \
  -e REEVE_FORWARD_PROXY \
  -e REEVE_NO_DESCRIPTION \
  -e HTTP_PROXY -e http_proxy \
  -e HTTPS_PROXY -e https_proxy \
  -e FTP_PROXY -e ftp_proxy \
  -e NO_PROXY -e no_proxy \
  -e ALL_PROXY -e all_proxy \
  --name reeve-runner-$(cat /proc/sys/kernel/random/uuid) \
  $REEVE_RUNNER_IMAGE
