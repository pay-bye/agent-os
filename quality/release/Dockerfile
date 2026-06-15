FROM gcr.io/distroless/static-debian12:nonroot

COPY agent-os /usr/local/bin/agent-os

USER nonroot:nonroot
ENTRYPOINT ["/usr/local/bin/agent-os"]
