#!/usr/bin/env bash

guard_publish() {
  local tag="$1"
  if [[ -z "$tag" ]]; then
    echo "release tag is required" >&2
    return 2
  fi
  require_protected_tag "$tag"
  require_destinations
  require_version_evidence "$tag"
  echo "publish eligible: $tag"
}

require_protected_tag() {
  local tag="$1"
  if [[ ! "$tag" =~ ^agent-os/v[0-9]+\.[0-9]+\.[0-9]+(-rc\.[0-9]+)?$ ]]; then
    echo "release tag must match agent-os/vMAJOR.MINOR.PATCH or release candidate form" >&2
    return 1
  fi
  if [[ "${GITHUB_REF_PROTECTED:-}" != "true" ]]; then
    echo "release tag must be protected" >&2
    return 1
  fi
}

require_destinations() {
  if [[ "${PUBLIC_RELEASE_DESTINATIONS_READY:-}" != "true" ]]; then
    echo "public release destinations are required" >&2
    return 1
  fi
}

require_version_evidence() {
  local version="${1#agent-os/}"
  if [[ "$version" =~ -rc\.[0-9]+$ ]]; then
    return 0
  fi
  if [[ -z "${PARITY_PROOF_URI:-}" ]]; then
    echo "stable release requires parity proof evidence" >&2
    return 1
  fi
  if [[ "$version" == v1.0.0 ]]; then
    require_v1_evidence
  fi
}

require_v1_evidence() {
  if [[ -z "${PRODUCTION_EVIDENCE_URI:-}" || -z "${DRIVER_EVIDENCE_URI:-}" ]]; then
    echo "v1 release requires production and driver evidence" >&2
    return 1
  fi
}
