# Build KAR
FROM golang:1.15 as builder

WORKDIR /kar/core

COPY go.mod go.mod
COPY go.sum go.sum
RUN go mod download

COPY pkg pkg
COPY internal internal
COPY cmd cmd
RUN CGO_ENABLED=0 go install ./...

# Create final image with desired executable
FROM alpine:3.11
ARG KAR_BINARY

COPY --from=builder /go/bin/$KAR_BINARY /kar/bin/$KAR_BINARY
