.PHONY: all deps gometalinter test cover

all: gometalinter test cover

deps:
	go get -u github.com/alecthomas/gometalinter
	gometalinter --install

gometalinter:
	gometalinter --vendor --deadline=1m --tests \
		--enable=gofmt \
		--enable=goimports \
		--enable=lll \
		--enable=misspell \
		--enable=unused

test:
	go test -v -race -cpu=1,2,4 -coverprofile=coverage.txt -covermode=atomic -benchmem -bench .

cover:
	go tool cover -html=coverage.txt -o coverage.html
