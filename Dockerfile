FROM golang:1.18 as builder

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY ./ ./

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o canipull main.go

FROM mcr.microsoft.com/aks/devinfra/base-os-runtime-static:master.220630.1
WORKDIR /
COPY --from=builder /workspace/canipull .

ENTRYPOINT ["/canipull"]
