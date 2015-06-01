#!/bin/bash

if [  $# -gt 0 ]
    then
    CONSUL_VERSION="$1";
    else
    CONSUL_VERSION="0.5.2";
fi

# install consul
wget "https://dl.bintray.com/mitchellh/consul/${CONSUL_VERSION}_linux_amd64.zip"
unzip "${CONSUL_VERSION}_linux_amd64.zip"

# check
./consul --version
