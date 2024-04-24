# docker build -t myapp . 
# docker run --rm -it -p 8080:8080 myapp:latest
FROM golang:latest AS builder
RUN apt-get update
ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64
WORKDIR /go/src/app
COPY go.mod .
RUN go mod download
# cache step
COPY . .
RUN go install

FROM scratch
COPY --from=builder /go/bin/myapp .
ENTRYPOINT ["./myapp"]