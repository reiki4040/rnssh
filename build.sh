#!/bin/bash

VERSION=0.4.0
HASH=$(git rev-parse --verify HEAD)
GOVERSION=$(go version)

go build -ldflags "-X main.version=$VERSION -X main.hash=$HASH -X \"main.goversion=$GOVERSION\""
