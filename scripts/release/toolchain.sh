#!/usr/bin/env bash

readonly GORELEASER_VERSION="v2.16.0"
readonly GO_VERSION="go1.26.4"

prepare_toolchain() {
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
  if [[ ! -x "$goroot/bin/go" ]]; then
    echo "$GO_VERSION is installed but not downloaded. Run: $GO_VERSION download" >&2
    return 1
  fi

  shim_dir="$(mktemp -d)"
  ln -s "$goroot/bin/go" "$shim_dir/go"
  export PATH="$shim_dir:$PATH"
  export GOTOOLCHAIN=local
  TOOLCHAIN_SHIM_DIR="$shim_dir"
  trap 'rm -rf "${TOOLCHAIN_SHIM_DIR:-}"' EXIT

  verify_toolchain
}

verify_toolchain() {
  local version
  version="$(go version)"
  if [[ "$version" != go\ version\ "$GO_VERSION"* ]]; then
    echo "go version mismatch: got '$version' want '$GO_VERSION'" >&2
    return 1
  fi
}

run_goreleaser() {
  curl -sfL https://goreleaser.com/static/run | \
    RELEASE_IMAGE_OWNER="${RELEASE_IMAGE_OWNER:-example}" \
    RELEASE_TAP_OWNER="${RELEASE_TAP_OWNER:-example}" \
    RELEASE_TAP_NAME="${RELEASE_TAP_NAME:-homebrew-tap}" \
    RELEASE_TAP_TOKEN="${RELEASE_TAP_TOKEN:-}" \
    DISTRIBUTION=pro \
    VERSION="$GORELEASER_VERSION" \
    bash -s -- "$@"
}
