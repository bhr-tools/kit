#!/bin/sh
# Browser tests of the kit with Playwright from nixpkgs: builds the demo
# server (cmd/demo), starts it on a throwaway database and drives it with
# Chromium, desktop and phone, dark and light. Screenshots: tests/e2e/out.
set -eu
cd "$(dirname "$0")/../.."
mkdir -p tests/e2e/out
work=$(mktemp -d)
trap 'kill $pid 2>/dev/null || true; rm -rf "$work"' EXIT

nix shell nixpkgs#go --command go build -o "$work/demo" ./cmd/demo
PORT=${PORT:-18099}
"$work/demo" -port "$PORT" -data-dir "$work/data" -no-auth >"$work/server.log" 2>&1 &
pid=$!
AUTH_PORT=$((PORT + 1))
"$work/demo" -port "$AUTH_PORT" -data-dir "$work/auth" >"$work/auth.log" 2>&1 &
pid="$pid $!"
trap 'kill $pid 2>/dev/null || true; rm -rf "$work"' EXIT

nix-shell -p 'python3.withPackages (ps: [ ps.playwright ])' --run "
  export PLAYWRIGHT_BROWSERS_PATH=$(nix build --no-link --print-out-paths nixpkgs#playwright-driver.browsers)
  export PLAYWRIGHT_SKIP_VALIDATE_HOST_REQUIREMENTS=true
  BASE_URL=http://127.0.0.1:$PORT AUTH_URL=http://127.0.0.1:$AUTH_PORT OUT=tests/e2e/out python3 tests/e2e/demo.py
"
