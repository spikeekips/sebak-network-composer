#!/bin/sh

set -e
set -x

env | sort

if [ ${SEBAK_INITIALIZE} -eq 1 ];then
    rm -rf $(echo $SEBAK_STORAGE | sed -e 's@file://@@g')/* || true
fi

/sebak genesis ${SEBAK_GENESIS_BLOCK} ${SEBAK_COMMON_ACCOUNT} || true

/sebak node \
    --network-id "${SEBAK_NETWORK_ID}" \
    $@
