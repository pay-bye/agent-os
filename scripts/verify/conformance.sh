#!/usr/bin/env bash

verify_conformance() {
  go test -count=1 ./tests/conformance/...
}
