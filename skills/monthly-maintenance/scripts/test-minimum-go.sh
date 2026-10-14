#!/usr/bin/env bash
# Run the unit tests with the lowest Go version that go.mod allows (its go directive), whatever Go
# is installed, by asking the go command to download and use that toolchain. Go's minimal version
# selection already builds with the module versions that go.mod requires, so the Go version is the
# range that the development environment can't show.
#
# usage, from the smarter-golang repository root:
#   skills/monthly-maintenance/scripts/test-minimum-go.sh
set -euo pipefail

MINIMUM=$(go mod edit -json | sed -n 's/.*"Go": "\([^"]*\)".*/\1/p')
if [ -z "$MINIMUM" ]; then
  echo "go.mod has no go directive" >&2
  exit 1
fi
echo "go.mod's minimum Go: $MINIMUM"
GOTOOLCHAIN="go$MINIMUM" go version
GOTOOLCHAIN="go$MINIMUM" go test -race ./...
