#!/bin/sh
set -eu
# Verification is offline; prepare missing modules explicitly before running.
export GOPROXY=off
export GOSUMDB=off
export GOTOOLCHAIN=local
cd "$(dirname "$0")/../.."
mkdir -p logs tmp/go-build tmp/go-tmp
repo_root="$PWD"
export GOCACHE="${GOCACHE:-$repo_root/tmp/go-build}"
export GOTMPDIR="${GOTMPDIR:-$repo_root/tmp/go-tmp}"
export TMPDIR="$GOTMPDIR"
cd services/api-server
go version > "$repo_root"/logs/api-server-environment.log
go env GOOS GOARCH GOMOD GOCACHE GOPATH >> "$repo_root"/logs/api-server-environment.log
go test -json ./... > "$repo_root"/logs/api-server-tests.jsonl 2>&1
go vet ./... > "$repo_root"/logs/api-server-vet.log 2>&1
go build ./... > "$repo_root"/logs/api-server-build-linux.log 2>&1
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build ./... > "$repo_root"/logs/api-server-build-windows.log 2>&1
printf 'Tests, vet, Linux build and Windows cross-build passed.\n'
