#!/usr/bin/env bash

run_snapshot() {
  run_goreleaser release --config "$CONFIG" --snapshot --clean --skip=publish
}
