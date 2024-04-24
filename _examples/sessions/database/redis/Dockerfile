FROM golang:latest AS builder
RUN apt-get update
WORKDIR /go/src/app
ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64
# Caching go modules and build the binary.
COPY go.mod .
RUN go mod download
COPY . .
RUN go install

FROM scratch
COPY --from=builder /go/bin/app .
ENTRYPOINT ["./app"]