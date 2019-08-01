#!/usr/bin/env bash
set -euxo pipefail

LOCATION=${1}

# shellcheck disable=SC1091
source .ci/scripts/install-go.sh

go get -u -d github.com/magefile/mage
pushd "$GOPATH/src/github.com/magefile/mage"
go run bootstrap.go

pushd "${LOCATION}"

## For debugging purposes
go env
mage -version
mage -l

## Package the metricbeats
mage package

## List the cached docker images
docker images
