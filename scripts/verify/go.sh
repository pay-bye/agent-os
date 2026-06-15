#!/usr/bin/env bash

readonly UNIT_LIMIT_SECONDS=300

verify_unit() {
  local started
  started="$(now)"

  run_step "$started" "toolchain version verification" 5 verify_toolchain
  run_step "$started" "gofmt clean check" 5 verify_format
  run_step "$started" "go vet" 30 go vet ./...
  run_step "$started" "staticcheck" 60 go run "$STATICCHECK_VERSION" ./...
  run_step "$started" "gosec" 60 go run "$GOSEC_VERSION" ./...
  run_step "$started" "govulncheck" 30 go run "$GOVULNCHECK_VERSION" ./...
  run_step "$started" "stepdown" 10 verify_stepdown
  run_step "$started" "protected path check" 5 verify_protected_paths
  run_step "$started" "git diff check" 5 git diff --check -- .
  run_step "$started" "unit tests" 60 go test -count=1 ./...
  run_step "$started" "unit race tests" 180 go test -race -count=1 ./...
  run_step "$started" "coverage floor check" 30 verify_coverage
  run_step "$started" "build verification" 30 go test -run '^$' ./...
}

run_step() {
  local aggregate_started="$1"
  local label="$2"
  local limit="$3"
  shift 3

  local started elapsed
  started="$(now)"
  "$@"
  elapsed="$(( "$(now)" - started ))"
  echo "$label: ${elapsed}s"
  if (( elapsed > limit )); then
    echo "step exceeded: $label elapsed=${elapsed}s ceiling=${limit}s" >&2
    return 1
  fi
  verify_unit_budget "$aggregate_started"
}

verify_unit_budget() {
  local started="$1"
  local elapsed
  elapsed="$(( "$(now)" - started ))"
  if (( elapsed >= UNIT_LIMIT_SECONDS )); then
    echo "unit gate exceeded: elapsed=${elapsed}s ceiling=${UNIT_LIMIT_SECONDS}s" >&2
    return 1
  fi
}

now() {
  date +%s
}

verify_toolchain() {
  local version
  version="$(go version)"
  if [[ "$version" != go\ version\ "$GO_VERSION"* ]]; then
    echo "go version mismatch: got '$version' want '$GO_VERSION'" >&2
    return 1
  fi

  go run "$STATICCHECK_VERSION" -version >/dev/null
  go run "$GOSEC_VERSION" -version >/dev/null
  go run "$GOVULNCHECK_VERSION" -version >/dev/null
}

verify_format() {
  local files
  local changed

  mapfile -d '' files < <(find . -name '*.go' -print0)
  if (( ${#files[@]} == 0 )); then
    return 0
  fi

  changed="$(gofmt -l "${files[@]}")"
  if [[ -n "$changed" ]]; then
    echo "gofmt changed files:" >&2
    echo "$changed" >&2
    return 1
  fi
}
