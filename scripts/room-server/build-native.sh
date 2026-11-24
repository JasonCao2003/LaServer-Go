#!/bin/sh
set -eu
if [ "$#" -ne 1 ] || [ ! -f "$1/include/enet/enet.h" ]; then
    printf '%s\n' 'Usage: sh scripts/room-server/build-native.sh /absolute/path/to/sky-enet-0.1.5/enet' >&2
    exit 1
fi
source_dir=$(cd "$1" && pwd)
cd "$(dirname "$0")/../.."
build_dir="$PWD/services/room-server/tmp/enet-bridge"
cmake -S services/room-server/internal/transport/native -B "$build_dir" -DSKYLA_ENET_SOURCE="$source_dir" -DCMAKE_BUILD_TYPE=Release
cmake --build "$build_dir" --parallel 2
printf 'Native build directory: %s\n' "$build_dir"
