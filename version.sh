#!/bin/sh
set -a
. $(dirname $0)/config.env
printf "%s" ${REEVE_VERSION:-development}
