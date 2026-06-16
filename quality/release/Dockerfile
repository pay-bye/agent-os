FROM gcr.io/distroless/static-debian12:nonroot

ARG TARGETOS
ARG TARGETARCH

COPY ${TARGETOS}/${TARGETARCH}/agent-os /usr/local/bin/agent-os

USER nonroot:nonroot
ENTRYPOINT ["/usr/local/bin/agent-os"]
