#!/bin/sh
set -a
. $(dirname $0)/.env
printf "%s" ${REEVE_VERSION:-development}
