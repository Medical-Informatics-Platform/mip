# Base images are pinned by digest; bump the digest to pick up base fixes.
FROM golang:1.27@sha256:5f1b98fa02fe396ebce2a0227a0e08309e11792915b9dd2e1f71053d7e7e6666 AS builder
ARG TARGETOS
ARG TARGETARCH
WORKDIR /workspace
# Dependencies first, so source changes reuse the module download layer.
COPY go.mod go.sum ./
RUN go mod download
COPY cmd/main.go cmd/main.go
COPY api/ api/
COPY internal/ internal/
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -a -o manager cmd/main.go

FROM gcr.io/distroless/static:nonroot@sha256:aafe993665bf6200db38a02c11a9351e782a84b6a45e7f33b6695638c88e3931
WORKDIR /
COPY --from=builder /workspace/manager .
USER 65532:65532
ENTRYPOINT ["/manager"]
