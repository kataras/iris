#!/bin/bash
# The script does automatic checking on a Go package and its sub-packages, including:
# 1. gofmt         (http://golang.org/cmd/gofmt/)
# 2. goimports     (http://golang.org/x/tools/cmd/goimports)
# 3. golint        (https://github.com/golang/lint)
# 4. go vet        (http://golang.org/cmd/vet)
# 5. ineffassign   (https://github.com/gordonklaus/ineffassign)
# 6. race detector (http://blog.golang.org/race-detector)
# 7. test coverage (http://blog.golang.org/cover)

go get golang.org/x/tools/cmd/goimports
go get github.com/golang/lint/golint
go get github.com/gordonklaus/ineffassign

set -e

# Automatic checks
echo gofmt check...
test -z "$(gofmt -l -w .     | tee /dev/stderr)"
echo goimports check...
test -z "$(goimports -l -w . | tee /dev/stderr)"
echo golint check...
test -z "$(golint .          | tee /dev/stderr)"
echo ineffassign check...
test -z "$(ineffassign .     | tee /dev/stderr)"

DIR_SOURCE="$(find . -maxdepth 10 -type f -not -path '*/vendor*' -name '*.go' | xargs -I {} dirname {} | sort | uniq)"

go vet ${DIR_SOURCE}
env GORACE="halt_on_error=1" go test -short -race ${DIR_SOURCE}


for dir in ${DIR_SOURCE};
do
    ineffassign $dir
done


# Run test coverage on each subdirectories and merge the coverage profile.

echo "mode: count" > profile.cov

for dir in ${DIR_SOURCE};
do
    go test -short -covermode=count -coverprofile=$dir/profile.tmp $dir
    if [ -f $dir/profile.tmp ]
    then
        cat $dir/profile.tmp | tail -n +2 >> profile.cov
        rm $dir/profile.tmp
    fi
done

go tool cover -func profile.cov

# To submit the test coverage result to coveralls.io,
# use goveralls (https://github.com/mattn/goveralls)

if [ -n "${CI_SERVICE+1}" ]; then
    echo "goveralls with" $CI_SERVICE
    if [ -n "${COVERALLS_TOKEN+1}" ]; then
        goveralls -coverprofile=profile.cov -service=$CI_SERVICE -repotoken $COVERALLS_TOKEN
    else
        goveralls -coverprofile=profile.cov -service=$CI_SERVICE
    fi
fi
