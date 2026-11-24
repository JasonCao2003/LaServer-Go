#!/bin/sh
set -eu
# Verification is offline; prepare missing modules explicitly before running.
export GOPROXY=off
export GOSUMDB=off
export GOTOOLCHAIN=local
cd "$(dirname "$0")/../.."
repo_root="$PWD"
mkdir -p logs tmp/go-build tmp/go-tmp services/room-server/dist
export GOCACHE="${GOCACHE:-$repo_root/tmp/go-build}"
export GOTMPDIR="${GOTMPDIR:-$repo_root/tmp/go-tmp}"
export TMPDIR="$GOTMPDIR"
cd services/room-server
if [ "$(go env GOOS)" != linux ] || [ "$(go env CGO_ENABLED)" != 1 ]; then
    printf '%s\n' 'Linux room-server verification requires GOOS=linux and CGO_ENABLED=1 with a C compiler.' >&2
    exit 1
fi
go version > "$repo_root/logs/room-server-environment.log"
go env GOOS GOARCH CGO_ENABLED CC GOMOD >> "$repo_root/logs/room-server-environment.log"
go test -count=1 -v ./... > "$repo_root/logs/room-server-tests.log" 2>&1
go vet ./... > "$repo_root/logs/room-server-vet.log" 2>&1
go build -o dist/room-server ./cmd/server > "$repo_root/logs/room-server-build-linux.log" 2>&1
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build ./... > "$repo_root/logs/room-server-build-windows.log" 2>&1
printf '%s\n' 'Room tests, vet, Linux build and Windows cross-build passed. See logs/ for optional integration skips.'
