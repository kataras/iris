FROM irisgo/cloud-native-go:latest

ENV APPSOURCES /go/src/github.com/iris-contrib/cloud-native-go

RUN ${APPSOURCES}/cloud-native-go