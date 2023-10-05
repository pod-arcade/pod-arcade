#!/bin/sh

mkdir -p bin/
GOOS=linux GOARCH=amd64 go build -buildvcs=false -o bin/ ./cmd/...