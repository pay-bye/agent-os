#!/usr/bin/env bash
set -euo pipefail

main() {
  cd "$(root)"
  load_verify_scripts
  prepare_go_toolchain

  case "${1:-}" in
    --unit)
      verify_unit
      ;;
    --integration)
      verify_integration
      ;;
    "")
      echo "missing mode: expected --unit or --integration" >&2
      return 2
      ;;
    *)
      echo "unknown flag: $1" >&2
      return 2
      ;;
  esac
}

root() {
  local script_dir
  script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
  cd "$script_dir/.." && pwd
}

load_verify_scripts() {
  source scripts/verify/toolchain.sh
  source scripts/verify/go.sh
  source scripts/verify/architecture.sh
  source scripts/verify/conformance.sh
  source scripts/verify/integration.sh
}

main "$@"
