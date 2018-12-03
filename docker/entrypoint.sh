#!/bin/sh

set -e
set -x

export SEBAK_STORAGE="file:///tmp/db/${SEBAK_NODE_ALIAS}"

env | sort

/sebak genesis ${SEBAK_GENESIS_BLOCK} ${SEBAK_COMMON_ACCOUNT} || true

/sebak node \
    --network-id "${SEBAK_NETWORK_ID}" \
    $@
