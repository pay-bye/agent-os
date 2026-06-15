# Governance

Agent OS maintainers own the runtime substrate, public contracts, release artifacts, and adopter
documentation in `github.com/pay-bye/agent-os`.

## Decision Ownership

- Runtime behavior is governed by code and tests in this repository.
- Invocation compatibility is governed by `contracts/invocation/v1.openapi.yaml`.
- Release compatibility is governed by release metadata and attestations published by this
  repository.
- Vocabulary compatibility is governed by `contracts/vocabulary/v1.schema.json`.

## Catalog Boundary

The future `github.com/pay-bye/agent-os-catalog` repository is discovery only. It links adopters to
component-owned contracts, manifests, releases, and docs. Catalog entries do not decide whether a
runtime, library, or driver is compatible.

## Release Authority

Public release authority begins when U6 publishes release artifacts and U8 accepts clean-machine
proof. Before those gates pass, command examples in this repository are target public paths, not
live adopter instructions.

## Changes

Governance changes are reviewed in the public repository. A governance change that alters
compatibility ownership updates the component-owned contract or release metadata in the same
logical change.
