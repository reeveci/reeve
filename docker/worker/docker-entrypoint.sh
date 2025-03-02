#!/bin/sh
set -e

if [ -n "$DOCKER_LOGIN_REGISTRY" ]; then
  if [ -z "$DOCKER_LOGIN_USER" ]; then
    echo Missing login user
    exit 1
  fi
  if [ -z "$DOCKER_LOGIN_PASSWORD" ]; then
    echo Missing login password
    exit 1
  fi

  echo Login attempt for $DOCKER_LOGIN_REGISTRY...
  printf "%s\n" "$DOCKER_LOGIN_PASSWORD" | docker login -u "$DOCKER_LOGIN_USER" --password-stdin $DOCKER_LOGIN_REGISTRY
fi

exec "$@"
