#!/bin/bash

VERSION=0.3.8
HASH=$(git rev-parse --verify HEAD)
BUILDDATE=$(date '+%Y/%m/%d %H:%M:%S %Z')

gom build -ldflags "-X main.version=$VERSION -X main.hash=$HASH -X \"main.builddate=$BUILDDATE\""
