# Install Agent OS

These examples use released public artifacts only. They require explicit configuration,
vocabulary, and verifier material at startup.

Public install commands become adopter-truth only after U3 rewrites public module paths, U6
publishes public release artifacts, and U8 accepts clean-machine proof. Until those gates pass,
these examples name the accepted public coordinates and command shapes without claiming that a
live public artifact exists.

Set the release coordinates used by the examples:

```sh
OWNER=pay-bye
REPO=agent-os
TAG=v0.1.0-rc.1
VERSION=${TAG#v}
```

## Archive

<!-- install:archive -->
```sh
ASSET_OS=linux
ASSET_ARCH=amd64
ASSET="agent-os_${VERSION}_${ASSET_OS}_${ASSET_ARCH}.tar.gz"
BASE_URL="https://github.com/${OWNER}/${REPO}/releases/download/${TAG}"

curl -fL -o "${ASSET}" "${BASE_URL}/${ASSET}"
curl -fL -o checksums.txt "${BASE_URL}/checksums.txt"
curl -fL -o checksums.txt.sigstore.json "${BASE_URL}/checksums.txt.sigstore.json"
cosign verify-blob --bundle checksums.txt.sigstore.json checksums.txt
gh attestation verify checksums.txt --owner "${OWNER}"
sha256sum -c checksums.txt --ignore-missing
tar -xzf "${ASSET}"
install agent-os /usr/local/bin/agent-os

agent-os serve \
  --config ./config.yaml \
  --from ./vocabulary.yaml \
  --verifier-file ./verifier.jwks
```

## Homebrew Cask

<!-- install:homebrew -->
```sh
brew tap "${OWNER}/tap"
brew install --cask agent-os
agent-os --help

agent-os serve \
  --config ./config.yaml \
  --from ./vocabulary.yaml \
  --verifier-file ./verifier.jwks
```

## GHCR

<!-- install:ghcr -->
```sh
IMAGE="ghcr.io/${OWNER}/agent-os:${VERSION}"
docker pull "${IMAGE}"
cosign verify "${IMAGE}"
gh attestation verify "oci://${IMAGE}" --owner "${OWNER}"

docker run --rm \
  -p 8080:8080 \
  -v "$PWD/config.yaml:/etc/agent-os/config.yaml:ro" \
  -v "$PWD/vocabulary.yaml:/etc/agent-os/vocabulary.yaml:ro" \
  -v "$PWD/verifier.jwks:/etc/agent-os/verifier.jwks:ro" \
  "${IMAGE}" serve \
    --config /etc/agent-os/config.yaml \
    --from /etc/agent-os/vocabulary.yaml \
    --verifier-file /etc/agent-os/verifier.jwks
```
