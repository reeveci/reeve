#!/bin/sh
set -e

if [ -z "$VERSION" ]; then
  echo "Missing VERSION variable"
  exit 1
fi

case "$VERSION" in
  v*)
    echo "VERSION should not include the leading 'v'"
    exit 1
esac

CURRENT_WORK_DIR=$(pwd)
cd $(dirname $0)/..

if OUT=$(git status --porcelain) && [ -n "$OUT" ]; then
  echo working directory is unclean
  cd $CURRENT_WORK_DIR
  exit 1
fi

CURRENT_BRANCH=$(git branch --show-current)
git checkout main

echo "REEVE_VERSION=$VERSION">config.env
scripts/cleanup.sh
git add . && git commit -m "release v$VERSION" ||:
git tag -a "v$VERSION" -m "v$VERSION"
git tag -a "reeve-cli/v$VERSION" -m "v$VERSION"
git tag -a "reeve-runner/v$VERSION" -m "v$VERSION"
git tag -a "reeve-server/v$VERSION" -m "v$VERSION"
git tag -a "reeve-worker/v$VERSION" -m "v$VERSION"
git push --follow-tags

git checkout $CURRENT_BRANCH

cd $CURRENT_WORK_DIR
