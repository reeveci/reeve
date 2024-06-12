#!/bin/sh

CURRENT_WORK_DIR=$(pwd)
cd $(dirname $0)/..

echo visiting reeve-server
cd reeve-server
go get github.com/reeveci/reeve-lib
go get -u ./...
go mod tidy
cd ..

echo visiting reeve-worker
cd reeve-worker
go get github.com/reeveci/reeve-lib
go get -u ./...
go mod tidy
cd ..

echo visiting reeve-runner
cd reeve-runner
go get github.com/reeveci/reeve-lib
go get -u ./...
go mod tidy
cd ..

echo visiting plugin-example
cd plugin-example
go get github.com/reeveci/reeve-lib
go get -u ./...
go mod tidy
cd ..

cd $CURRENT_WORK_DIR