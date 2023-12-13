#!/bin/sh
set -e

CURRENT_WORK_DIR=$(pwd)
cd $(dirname $0)

mkdir -p out

docker run --rm -dit -p 9443:9443 \
  -v $(pwd)/plugins:/etc/reeve/plugins \
  -v $(pwd)/cert:/cert \
  -e REEVE_HTTP_PORT= \
  -e REEVE_TLS_CERT_FILE=/cert/server.crt \
  -e REEVE_TLS_KEY_FILE=/cert/server.key \
  -e REEVE_MESSAGE_SECRETS=dGVzdDp0ZXN0 \
  -e REEVE_CLI_SECRETS=dGVzdDp0ZXN0 \
  -e REEVE_WORKER_SECRETS=dGVzdDp0ZXN0 \
  -e REEVE_SHARED_TASK_DOMAINS=trust \
  -e REEVE_SHARED_TRUSTED_DOMAINS=trust \
  -e REEVE_PLUGIN_EXAMPLE_ENABLED=true \
  -e REEVE_PLUGIN_EXAMPLE_FLAG=example \
  --name reeve-server reeve >/dev/null
docker logs -f reeve-server &

function spawn_worker() {
  docker run --rm -i -v /var/run/docker.sock:/var/run/docker.sock --link reeve-server \
    -e REEVE_SERVER_API=https://reeve-server:9443 \
    -e REEVE_WORKER_SECRET=dGVzdDp0ZXN0 \
    -e REEVE_RUNNER_IMAGE=reeve-runner \
    -v $(pwd)/cert/server.crt:/etc/ssl/certs/reeve-server.crt \
    --name $1 reeve-worker 2>&1 >out/$1.log &
}

spawn_worker reeve-worker1
spawn_worker reeve-worker2

sleep 1

./send-message.sh
./send-message.sh

cd $CURRENT_WORK_DIR

function kill_background() {
  docker stop reeve-server
}
trap kill_background SIGINT

docker attach reeve-server 2>&1 >/dev/null
