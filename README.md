# SEBAK Network Composer

## Installation

```sh
$ go get github.com/spikeekips/sebak-network-composer
```

## Usage

Build image first
```sh
$ cd docker
$ docker build -t boscoin/sebak-network-composer:latest .
```

```sh
$ sebak-network-composer \
    --force \
    --log-level debug \
    --sebak-log-level debug \
    --env "SEBAK_GENESIS_BLOCK=SBXBRFM4UDBHREM2XRM6IIOXNR52N6NAKWIMR7MR4XMNJ5VA4WC27QDY" \
```
