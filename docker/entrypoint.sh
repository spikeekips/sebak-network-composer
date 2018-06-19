#!/bin/sh

set -e
set -x

export SEBAK_ENDPOINT="${SEBAK_ENDPOINT}?HTTP2LogOutput=/tmp/http-${SEBAK_NODE_ALIAS}.log"
export SEBAK_STORAGE=file:///tmp/db/${SEBAK_NODE_ALIAS}

env | sort

cd /go/src/boscoin.io/sebak
go run cmd/sebak/main.go genesis ${SEBAK_GENESIS_BLOCK} --balance 1000000000000000000 || true

for v in ${SEBAK_VALIDATORS}; do
    VALIDATOR_ARGS="${VALIDATOR_ARGS} --validator=${v}"
done

go run cmd/sebak/main.go node --network-id "${SEBAK_NETWORK_ID}" --secret-seed ${SEBAK_SECRET_SEED} --log-level debug ${VALIDATOR_ARGS} $@
