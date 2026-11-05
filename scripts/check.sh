#!/usr/bin/env bash
# Runs locally what .github/workflows/ci.yml ran on a pull request:
# golangci-lint (with its formatters), build, vet and the tests. GitHub
# Actions CI is off, so this is the check a change gets. The tests are fast,
# so there is no --full mode.
set -euo pipefail
cd "$(dirname "$0")/.."

if [ $# -gt 0 ]; then echo "usage: scripts/check.sh" >&2; exit 2; fi

# The version ci.yml pinned and .golangci.yml is written against.
golangci_version=v2.13.2

step() { printf '\n==> %s\n' "$*"; }
need() {
  command -v "$1" >/dev/null 2>&1 || { echo "check: $1 is not installed. $2" >&2; exit 1; }
}

need go "Install Go from https://go.dev/dl/ (the version in go.mod)."

# CI ran the Go of go.mod (setup-go with go-version-file). A newer local Go
# breaks golangci-lint $golangci_version, whose type checker cannot read its
# export data, so every step uses that toolchain; go downloads it once.
export GOTOOLCHAIN="go$(sed -n 's/^go //p' go.mod)"

# CI built golangci-lint from source with the Go of go.mod (install-mode:
# goinstall). go run does the same and caches the build.
step "golangci-lint run ($golangci_version)"
go run "github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$golangci_version" run --allow-serial-runners ./...

step "go build ./..."
go build ./...
step "go vet ./..."
go vet ./...
step "go test ./..."
go test ./...

printf '\ncheck: all steps passed\n'
