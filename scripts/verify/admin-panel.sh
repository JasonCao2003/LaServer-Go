#!/bin/sh
set -eu
cd "$(dirname "$0")/../.."
repo_root="$PWD"
mkdir -p logs tmp/go-build tmp/go-tmp
export GOPROXY=off GOTOOLCHAIN=local
export GOCACHE="${GOCACHE:-$repo_root/tmp/go-build}"
export GOTMPDIR="${GOTMPDIR:-$repo_root/tmp/go-tmp}"
export TMPDIR="$GOTMPDIR"
cd services/admin-panel
go test -count=1 -json ./... > "$repo_root/logs/admin-panel-test.jsonl" 2>&1
go test -race -count=1 -json ./... > "$repo_root/logs/admin-panel-race.jsonl" 2>&1
go vet ./... > "$repo_root/logs/admin-panel-vet.log" 2>&1
go build ./... > "$repo_root/logs/admin-panel-build.log" 2>&1
printf '%s\n' 'Admin panel tests, race, vet and build passed.'
