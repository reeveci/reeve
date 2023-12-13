#!/bin/bash
set -e

CURRENT_WORK_DIR=$(pwd)
cd $(dirname $0)

mkdir -p cert
cd cert

# Create CA key and cert
openssl genrsa -out ca.key 2048
openssl req -new -x509 -days 365 -key ca.key -subj "/O=ReeveCI/CN=Reeve Root CA" -out ca.crt

# Create CSR
openssl req -newkey rsa:2048 -nodes -keyout server.key -subj "/O=ReeveCI/CN=reeve-server" -out server.csr

# Sign certificate
openssl x509 -req -extfile <(printf "subjectAltName=DNS:reeve-server") -days 365 -in server.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out server.crt

cd $CURRENT_WORK_DIR
