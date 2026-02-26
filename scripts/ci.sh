#!/usr/bin/env bash
set -euo pipefail

echo "=== build ==="
go build ./...

echo "=== vet ==="
go vet ./...

echo "=== test ==="
go test -race ./...

echo "=== gosec ==="
gosec -exclude=G104,G706 ./...

echo "=== staticcheck ==="
staticcheck ./...

echo "=== govulncheck ==="
govulncheck ./...

echo "=== all checks passed ==="
