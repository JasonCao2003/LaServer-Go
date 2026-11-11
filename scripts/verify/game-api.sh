#!/bin/sh
set -eu
cd "$(dirname "$0")/../.."
export PATH="$HOME/.local/opt/go1.27.4/bin:$PATH"
mkdir -p logs
go version > logs/game-api-environment.log
go env GOOS GOARCH GOMOD GOCACHE GOPATH >> logs/game-api-environment.log
go test -json ./... > logs/game-api-tests.jsonl 2>&1
go vet ./... > logs/game-api-vet.log 2>&1
go build ./... > logs/game-api-build-linux.log 2>&1
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build ./... > logs/game-api-build-windows.log 2>&1
printf 'Tests, vet, Linux build and Windows cross-build passed.\n'
