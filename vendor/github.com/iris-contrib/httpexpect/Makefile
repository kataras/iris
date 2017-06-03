all: update test check

update:
	go get -u -t . ./_examples

test:
	go test . ./_examples

check:
	go get -u github.com/alecthomas/gometalinter
	gometalinter --install
	gometalinter --config .gometalinter

fmt:
	gofmt -s -w . ./_examples
