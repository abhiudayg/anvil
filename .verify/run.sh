#!/bin/sh
# agstack phased verifier: doctor -> unit -> static -> functional.
# Artifacts land in .verify/artifacts/<feature>.<phase>.log
set -u
FEATURE="${1:-tui-panes}"
ART=".verify/artifacts"
mkdir -p "$ART"
fail=0

phase() {
    name="$1"; shift
    echo "== $name: $*"
    if "$@" >"$ART/$FEATURE.$name.log" 2>&1; then
        echo "   PASS"
    else
        echo "   FAIL (see $ART/$FEATURE.$name.log)"
        fail=1
    fi
}

phase doctor     go build ./...
phase unit       go test ./internal/tui ./internal/guard
phase static     go vet ./internal/tui ./cmd/anvil
phase functional go test -run Functional -v ./internal/tui
phase cli        go test ./cmd/anvil

exit "$fail"
