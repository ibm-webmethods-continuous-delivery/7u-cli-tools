# syntax=docker/dockerfile:1
# Shared by every tool. Build context is code/<tool>/ (see docker-compose.yml).
# Result: a scratch image holding one static binary at /<tool>
# (plus /SKILL.md when the tool ships one).

FROM golang:1-alpine AS builder

ARG TOOL
ARG VERSION=dev
ARG BUILD_DATE=unknown
ARG GIT_COMMIT=unknown
# Go package that declares the string vars version, buildDate, gitCommit.
# "main" by default; e.g. "github.com/org/repo/cmd" when they live elsewhere.
ARG VERSION_PKG=main

WORKDIR /src

COPY go.mod go.sum* ./
RUN go mod download

COPY . .
RUN mkdir /out \
    && CGO_ENABLED=0 GOOS=linux go build -trimpath \
      -ldflags "-s -w \
        -X ${VERSION_PKG}.version=${VERSION} \
        -X ${VERSION_PKG}.buildDate=${BUILD_DATE} \
        -X ${VERSION_PKG}.gitCommit=${GIT_COMMIT}" \
      -o /out/${TOOL} ./cmd/${TOOL} \
    && if [ -f SKILL.md ]; then cp SKILL.md /out/SKILL.md; fi

# Fail the build if the binary is not statically linked.
RUN ldd /out/${TOOL} 2>&1 | grep -qE "(statically linked|Not a valid dynamic program|not a dynamic executable)" \
    || (echo "ERROR: ${TOOL} is not statically linked" && exit 1)

FROM scratch

ARG TOOL
ARG VERSION=dev
ARG GIT_COMMIT=unknown

LABEL org.opencontainers.image.title="${TOOL}" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.revision="${GIT_COMMIT}" \
      org.opencontainers.image.source="https://github.com/ibm-webmethods-continuous-delivery/7u-cli-tools"

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /out/ /
