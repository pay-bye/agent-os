#!/usr/bin/env bash

readonly CONFIG="quality/release/goreleaser.yaml"

check_config() {
  run_goreleaser check --config "$CONFIG"
}
