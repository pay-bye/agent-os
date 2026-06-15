#!/usr/bin/env bash

readonly INTEGRATION_LIMIT_SECONDS=120
readonly INTEGRATION_ENV_FILE="company/control-plane/control-plane.env.local"
readonly INTEGRATION_ENV_VARIABLE="CONTROL_PLANE_TEST_DATABASE_URL"

verify_integration() {
  load_integration_env

  local started elapsed
  started="$(now)"
  verify_conformance
  go test -tags=integration -count=1 ./tests/integration/...
  elapsed="$(( "$(now)" - started ))"
  echo "integration tests: ${elapsed}s"
  if (( elapsed > INTEGRATION_LIMIT_SECONDS )); then
    echo "integration gate exceeded: elapsed=${elapsed}s ceiling=${INTEGRATION_LIMIT_SECONDS}s" >&2
    return 1
  fi
}

load_integration_env() {
  local env_file
  if ! env_file="$(integration_env_file)"; then
    print_integration_env_pointer
    return 1
  fi

  set -a
  source "$env_file"
  set +a

  if [[ -z "${CONTROL_PLANE_TEST_DATABASE_URL:-}" ]]; then
    print_integration_env_pointer
    return 1
  fi

  export DATABASE_URL="$CONTROL_PLANE_TEST_DATABASE_URL"
}

integration_env_file() {
  local dir
  dir="$(root)"

  while true; do
    if [[ -f "$dir/$INTEGRATION_ENV_FILE" ]]; then
      printf '%s\n' "$dir/$INTEGRATION_ENV_FILE"
      return 0
    fi

    local parent
    parent="$(dirname "$dir")"
    if [[ "$parent" == "$dir" ]]; then
      return 1
    fi
    dir="$parent"
  done
}

print_integration_env_pointer() {
  {
    echo "DATABASE_URL is required for integration tests"
    echo "Expected env file: $INTEGRATION_ENV_FILE"
    echo "Expected variable: $INTEGRATION_ENV_VARIABLE"
    echo "Required source/export command:"
    echo "  set -a; source $INTEGRATION_ENV_FILE; set +a"
    echo '  export DATABASE_URL="${CONTROL_PLANE_TEST_DATABASE_URL:?}"'
  } >&2
}
