# Agent OS

Agent OS is the substrate runtime for agent work execution. It owns the kernel, transport,
storage contracts, vocabulary schema, executable, release metadata, and public runtime docs.

## Public Coordinates

Future public repository: `github.com/pay-bye/agent-os`

Future public module: `github.com/pay-bye/agent-os`

Public install commands become adopter-truth only after U3 rewrites public module paths, U6
publishes public release artifacts, and U8 accepts clean-machine proof. Until those gates pass,
docs name the public coordinates and command shapes without claiming that a live public artifact
exists.

## Documents

- [Install](docs/install.md)
- [Update](docs/update.md)
- [Governance](docs/governance.md)
- [Catalog Cross-Reference](docs/catalog.md)
- [Contributing](CONTRIBUTING.md)
- [Security](SECURITY.md)
- [Code of Conduct](CODE_OF_CONDUCT.md)

## Component-Owned Truth

Compatibility evidence is owned by this repository's contracts and release metadata:

- `contracts/invocation/v1.openapi.yaml`
- `contracts/release/v1.schema.json`
- `contracts/vocabulary/v1.schema.json`
- GitHub release artifacts and attestations published from this repository

The future `github.com/pay-bye/agent-os-catalog` repository is discovery only. Catalog entries
point to component-owned contracts, manifests, and release metadata; they do not decide
compatibility.

## Verification

Run the unit gate from the repository root:

```sh
GOTOOLCHAIN=local bash scripts/verify.sh --unit
```
