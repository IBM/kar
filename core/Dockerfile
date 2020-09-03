# Build KAR
FROM golang:1.15 as builder

RUN mkdir -p $GOPATH/src/github.ibm.com/solsa/kar
WORKDIR $GOPATH/src/github.ibm.com/solsa/kar

COPY cmd cmd
COPY pkg pkg
COPY internal internal
COPY go.mod go.mod
COPY go.sum go.sum

RUN CGO_ENABLED=0 go install ./...

# Create final image with desired executable
FROM alpine:3.11
ARG KAR_BINARY

RUN mkdir -p /kar
COPY --from=builder /go/bin/$KAR_BINARY /kar

