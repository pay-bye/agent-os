#!/usr/bin/env bash

write_subjects() {
  local subjects
  subjects="$(subject_lines)"
  if [[ -z "$subjects" ]]; then
    echo "release subjects are empty" >&2
    return 1
  fi
  printf 'base64=%s\n' "$(printf '%s\n' "$subjects" | base64 | tr -d '\n')"
}

subject_lines() {
  append_checksum_subjects
  append_image_subjects
}

append_checksum_subjects() {
  local path="dist/release/checksums.txt"
  require_subject_file "$path"
  cat "$path"
}

append_image_subjects() {
  local path="dist/release/digests.txt"
  local line

  require_subject_file "$path"
  while IFS= read -r line || [[ -n "$line" ]]; do
    normalized_image_subject "$line"
  done < "$path"
}

normalized_image_subject() {
  local line="$1"
  local first
  local second

  if [[ -z "${line//[[:space:]]/}" ]]; then
    return 0
  fi

  read -r first second _ <<< "$line"
  if [[ -z "${second:-}" ]]; then
    subject_from_digest_reference "$first"
    return
  fi

  if [[ "$first" == sha256:* ]]; then
    printf '%s  %s\n' "${first#sha256:}" "$second"
    return
  fi
  if [[ "$second" == sha256:* ]]; then
    printf '%s  %s\n' "${second#sha256:}" "$first"
    return
  fi
  printf '%s  %s\n' "$first" "$second"
}

subject_from_digest_reference() {
  local reference="$1"
  if [[ "$reference" =~ ^(.+)@sha256:([0-9a-fA-F]{64})$ ]]; then
    printf '%s  %s\n' "${BASH_REMATCH[2]}" "$reference"
    return
  fi
  echo "invalid image digest line: $reference" >&2
  return 1
}

require_subject_file() {
  local path="$1"
  if [[ ! -s "$path" ]]; then
    echo "release subject file is required: $path" >&2
    return 1
  fi
}
