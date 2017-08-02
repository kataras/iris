FROM golang:1.8.3-alpine

RUN apk update && apk upgrade && apk add --no-cache bash git
RUN go get github.com/iris-contrib/cloud-native-go

ENV SOURCES /go/src/github.com/iris-contrib/cloud-native-go
# COPY . ${SOURCES}

RUN cd ${SOURCES} $$ CGO_ENABLED=0 go build

ENTRYPOINT cloud-native-go
EXPOSE 8080