FROM golang:alpine AS builder
LABEL maintainer="BOSCoin Developers <devteam@boscoin.io>"

ENV LC_ALL=en_US.UTF-8
ENV LANG=en_US.UTF-8
ENV TERM=xterm
ENV DEBIAN_FRONTEND=noninteractive
ENV DEBIAN_FRONTEND=noninteractive

RUN ( \
    apk --no-cache add git gcc musl-dev; \
)

WORKDIR /sebak
RUN git clone https://github.com/bosnet/sebak /sebak

RUN go build boscoin.io/sebak/cmd/sebak

################################################################################
FROM golang:alpine as final

COPY --from=builder /sebak/sebak /sebak
Add ./sebak.crt /sebak.crt
Add ./sebak.key /sebak.key

ADD ./entrypoint.sh /entrypoint.sh

RUN ( \
    chmod 755 /entrypoint.sh; \
    mkdir -p /root/.ssh; \
    chmod 700 /root/.ssh; \
    echo 'StrictHostKeyChecking=no' >> ~/.ssh/config; \
    echo 'alias ls="ls --color=auto"' > ~/.bashrc; \
    echo 'alias v="ls -al"' >> ~/.bashrc; \
    echo 'alias a="ls -a"' >> ~/.bashrc; \
)

ENTRYPOINT ./entrypoint.sh

# vim: set filetype=Dockerfile:
