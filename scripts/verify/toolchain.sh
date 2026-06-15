#!/usr/bin/env bash

readonly GO_VERSION="go1.26.4"
readonly STATICCHECK_VERSION="honnef.co/go/tools/cmd/staticcheck@v0.7.0"
readonly GOSEC_VERSION="github.com/securego/gosec/v2/cmd/gosec@v2.26.1"
readonly GOVULNCHECK_VERSION="golang.org/x/vuln/cmd/govulncheck@v1.3.0"
readonly STEPDOWN_PACKAGE="stepdown.dev/go/cmd/stepdown@v0.1.1"
readonly EXPECTED_STEPDOWN_PACKAGE="stepdown.dev/go/cmd/stepdown@v0.1.1"

prepare_go_toolchain() {
  local base_go
  local gopath_bin
  local goroot
  local shim_dir

  base_go="$(command -v go || true)"
  if [[ -n "$base_go" ]]; then
    gopath_bin="$("$base_go" env GOPATH)/bin"
    export PATH="$gopath_bin:$PATH"
  fi

  if ! command -v "$GO_VERSION" >/dev/null 2>&1; then
    echo "$GO_VERSION is required. Install with: GOTOOLCHAIN=local go install golang.org/dl/$GO_VERSION@latest && $GO_VERSION download" >&2
    return 1
  fi

  goroot="$("$GO_VERSION" env GOROOT)"
  if [[ ! -x "$goroot/bin/go" || ! -x "$goroot/bin/gofmt" ]]; then
    echo "$GO_VERSION is installed but not downloaded. Run: $GO_VERSION download" >&2
    return 1
  fi

  shim_dir="$(mktemp -d)"
  ln -s "$goroot/bin/go" "$shim_dir/go"
  ln -s "$goroot/bin/gofmt" "$shim_dir/gofmt"
  export PATH="$shim_dir:$PATH"
  export GOTOOLCHAIN=local
  TOOLCHAIN_SHIM_DIR="$shim_dir"
  trap 'rm -rf "${TOOLCHAIN_SHIM_DIR:-}"' EXIT

  verify_toolchain
}

verify_stepdown() {
  verify_stepdown_pin
  go run "$STEPDOWN_PACKAGE" ./...
}

verify_stepdown_pin() {
  if [[ "$STEPDOWN_PACKAGE" != "$EXPECTED_STEPDOWN_PACKAGE" ]]; then
    echo "stepdown version mismatch: got '$STEPDOWN_PACKAGE' want '$EXPECTED_STEPDOWN_PACKAGE'" >&2
    return 1
  fi
}
