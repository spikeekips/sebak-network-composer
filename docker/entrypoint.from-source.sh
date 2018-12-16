#!/bin/sh

set -e
set -x

env | sort

if [ ${SEBAK_INITIALIZE} -eq 1 ];then
    rm -rf $(echo $SEBAK_STORAGE | sed -e 's@file://@@g')/* || true
fi

cd /sebak
go run cmd/sebak/main.go genesis ${SEBAK_GENESIS_BLOCK} ${SEBAK_COMMON_ACCOUNT} || true

go run cmd/sebak/main.go node \
    --network-id "${SEBAK_NETWORK_ID}" \
    $@
