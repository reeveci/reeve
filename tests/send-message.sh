#!/bin/sh

curl -X POST -k -H "Content-Type: application/json" -H "Authorization: Bearer dGVzdDp0ZXN0" -d '"test"' "https://localhost:9443/api/v1/message?target=example"
