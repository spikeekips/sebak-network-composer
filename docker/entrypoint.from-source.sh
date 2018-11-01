#!/bin/sh

set -e
set -x

export SEBAK_STORAGE="file:///tmp/db/${SEBAK_NODE_ALIAS}"

env | sort

cd /sebak
go run cmd/sebak/main.go genesis ${SEBAK_GENESIS_BLOCK} ${SEBAK_COMMON_ACCOUNT} --balance 1000000000000000000 || true

go run cmd/sebak/main.go node \
    --network-id "${SEBAK_NETWORK_ID}" \
    $@
