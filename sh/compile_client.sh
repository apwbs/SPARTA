#!/usr/bin/env bash
set -euo pipefail

cd ../src/client
CGO_CFLAGS=-I/opt/ego/include CGO_LDFLAGS=-L/opt/ego/lib ego-go build client.go