#!/usr/bin/env bash
set -euo pipefail

main() {
  cd "$(root)"
  load_release_scripts

  case "${1:-}" in
    --config)
      prepare_toolchain
      check_config
      ;;
    --snapshot)
      prepare_toolchain
      run_snapshot
      ;;
    --guard)
      guard_publish "${2:-}"
      ;;
    --subjects)
      write_subjects
      ;;
    "")
      echo "missing mode: expected --config, --snapshot, --guard, or --subjects" >&2
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

load_release_scripts() {
  source scripts/release/toolchain.sh
  source scripts/release/config_check.sh
  source scripts/release/snapshot.sh
  source scripts/release/publish_guard.sh
  source scripts/release/subjects.sh
}

main "$@"
