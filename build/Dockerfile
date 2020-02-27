FROM golang:1.13 as builder

# Build KAR sidecar
RUN mkdir /kar
WORKDIR /kar
COPY cmd cmd
COPY go.mod go.mod
COPY go.sum go.sum
RUN go install ./...

# Create final image with built executables
FROM alpine:3.11

RUN mkdir -p /kar
COPY --from=builder /go/bin/kar /kar

