#!/bin/bash

# TODO: We should not build octopus at that directory.
cd anchor-ipam; GOOS=linux go build
cd ..; cp -r octopus ../../containernetworking/plugins/plugins/main
cd ../../containernetworking/plugins && ./build.sh
cd -; cp ../../containernetworking/plugins/bin/octopus anchor-ipam
cd anchor-ipam && docker build -t anchor:v0.3.1 .
