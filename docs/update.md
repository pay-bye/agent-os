# Update Agent OS

Updates use public release artifacts from `github.com/pay-bye/agent-os`.

Public update commands become adopter-truth only after U6 publishes release artifacts and U8
accepts clean-machine proof. Before those gates pass, the commands below document the release path
shape only.

## Archive

```sh
OWNER=pay-bye
REPO=agent-os
TAG=v0.1.0-rc.1
VERSION=${TAG#v}
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
agent-os --version
```

## Container

```sh
OWNER=pay-bye
TAG=v0.1.0-rc.1
VERSION=${TAG#v}
IMAGE="ghcr.io/${OWNER}/agent-os:${VERSION}"

docker pull "${IMAGE}"
cosign verify "${IMAGE}"
gh attestation verify "oci://${IMAGE}" --owner "${OWNER}"
```

Compatibility is determined by this repository's release metadata and contracts. The catalog links
to those sources; it does not decide whether an update is compatible.
