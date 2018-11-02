# SEBAK Network Composer

## Installation

```sh
$ go get github.com/spikeekips/sebak-network-composer
```

## Usage

Build image for `sebak` executable mode.
```sh
$ sebak-network-composer build
```
By default, this will create new image `boscoin/sebak-network-composer:latest` docker image.

Build image for `sebak` source mode.
```sh
$ sebak-network-composer build --source
```
By default, this will create new image `boscoin/sebak-network-composer:latest-source` docker image.

### Run Nodes

```sh
$ sebak-network-composer run \
    --force \
    --log-level debug \
    --sebak-log-level debug \
    --n 4 \
    config.toml
```

This will read the configuration from `config.toml` and deploy nodes.


### Download Docker Logs

```sh
$ sebak-network-composer  logs config.toml --ouput-directory /tmp/
```

### Node Info

```
$ sebak-network-composer node-info config.toml  | jq -r '[.node.endpoint, .block.height] | "endpoint=\(.[0]) block-height=\(.[1])"'
https://172.31.22.130:12001 161
https://172.31.22.130:12000 161
https://172.31.25.219:12001 161
https://172.31.25.219:12000 161
```

```
$ sebak-network-composer node-info config.toml  --verbose | pbcopy
```
```json
{
  "node": {
    "version": {
      "version": "",
      "git-commit": "",
      "git-state": "",
      "build-date": ""
    },
    "state": "CONSENSUS",
    "alias": "GDXJ.3YI5",
    "address": "GDXJZRZXEMB7WTRKVVOZZ4JGIUJPN627M3UXYM5VB3WQVHF53YI5Z6TK",
    "endpoint": "https://172.31.22.130:12001",
    "validators": {
      "GBF4PFHGZ6ASJAFKOSNYTC7EJOPV3PLPXF46GOYAFRQQYAI5LUP7HOPD": {
        "address": "GBF4PFHGZ6ASJAFKOSNYTC7EJOPV3PLPXF46GOYAFRQQYAI5LUP7HOPD",
        "alias": "GBF4.LUP7",
        "endpoint": "https://172.31.22.130:12000"
      },
      "GD6DTWQZWBHA463BSAI26WDN45H2LE4X6B6NEU6EQH2A2RWDQDZTFWTI": {
        "address": "GD6DTWQZWBHA463BSAI26WDN45H2LE4X6B6NEU6EQH2A2RWDQDZTFWTI",
        "alias": "GD6D.QDZT",
        "endpoint": "https://172.31.25.219:12000"
      },
      "GDO7BUHC24KVHW3WJLVZ4TBKMBGPK74N37YJHU4AWHAVCUDKKCU3M7JS": {
        "address": "GDO7BUHC24KVHW3WJLVZ4TBKMBGPK74N37YJHU4AWHAVCUDKKCU3M7JS",
        "alias": "GDO7.KCU3",
        "endpoint": "https://172.31.25.219:12001"
      },
      "GDXJZRZXEMB7WTRKVVOZZ4JGIUJPN627M3UXYM5VB3WQVHF53YI5Z6TK": {
        "address": "GDXJZRZXEMB7WTRKVVOZZ4JGIUJPN627M3UXYM5VB3WQVHF53YI5Z6TK",
        "alias": "GDXJ.3YI5",
        "endpoint": "https://172.31.22.130:12001"
      }
    }
  },
  "policy": {
    "network-id": "test sebak-network",
    "initial-balance": "1000000000000000000",
    "base-reserve": "1000000",
    "base-fee": "10000",
    "block-time": 5000000000,
    "operations-limit": 1000,
    "transactions-limit": 1000,
    "genesis-block-confirmed-time": "2018-04-17T5:07:31.000000000Z",
    "inflation-ratio": "0.00000010000000000",
    "block-height-end-of-inflation": 36000000
  },
  "block": {
    "height": 180,
    "hash": "5byobNfpjJg77c5EWH2F2Eu6L2EG1SwCNNGBGwBAgcMT",
    "total-txs": 180,
    "total-ops": 0
  }
}
...
```

### Configuration File

```toml
docker-path = "./docker"

# SCRRCYK5IFL23GEIIXW5MPXHSBX5O3KQEFT6RXOXXANQ3S6DL6YJ7P27
# GAPYEQH7MC5SGA7MLLWOKBEXFXPOM34APVQRW6OWBDMH5G3KDRJ66IQQ
genesis = "GAPYEQH7MC5SGA7MLLWOKBEXFXPOM34APVQRW6OWBDMH5G3KDRJ66IQQ"

[hosts]
  [hosts.seoul0]
  host = "tcp://54.180.8.229:2376"
  cert = "~/.docker/machine/machines/ex-seoul0"
  volume = [
    "/home/ubuntu/sebak/take-1-not-save-operation:/sebak"
  ]
  env = [
    "SEBAK_RATE_LIMIT_API=0-s",
    "SEBAK_RATE_LIMIT_NODE=0-s"
  ]

  [hosts.seoul1]
  host = "tcp://52.79.243.48:2376"
  cert = "~/.docker/machine/machines/ex-seoul1"
  volume = [
    "/home/ubuntu/sebak/take-1-not-save-operation:/sebak"
  ]
  env = [
    "SEBAK_RATE_LIMIT_API=0-s",
    "SEBAK_RATE_LIMIT_NODE=0-s"
  ]
```

* `docker-path`: the base path for building docker image.
* `genesis`: the public address of genesis account
* `hosts`: the docker host to deploy nodes
* `hosts.<host name>`: set the host name
* `host`: host address to connect
* `cert`: cert files for docker client
* `volume`: set the mount volumes for docker container
* `env`: set the environmental variables for docker container

