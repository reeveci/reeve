#!/bin/sh

CURRENT_WORK_DIR=$(pwd)
cd $(dirname $0)/../plugin-example

GOOS=linux go build -o ../tests/plugins/example .

cd $CURRENT_WORK_DIR
