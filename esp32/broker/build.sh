#!/bin/sh
set -eu
cd "$(dirname "$0")"

: "${GOOS:=linux}"
: "${GOARCH:=arm64}"
: "${CGO_ENABLED:=0}"
export GOOS GOARCH CGO_ENABLED

mkdir -p dist

go test ./...
go build -trimpath -ldflags="-s -w" -o dist/esp32d ./cmd/esp32d
go build -trimpath -ldflags="-s -w" -o dist/esp32ctl-broker ./cmd/esp32ctl

printf 'built %s/%s\n' "$GOOS" "$GOARCH"
