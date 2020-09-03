# Build KAR
FROM golang:1.15 as builder

RUN mkdir -p /kar/core
WORKDIR /kar/core

COPY go.mod go.sum ./
RUN go mod download

COPY pkg pkg
COPY internal internal
COPY cmd cmd
RUN CGO_ENABLED=0 go install ./...

# Create final image with desired executable
FROM alpine:3.11
ARG KAR_BINARY

RUN mkdir -p /kar
COPY --from=builder /go/bin/$KAR_BINARY /kar
