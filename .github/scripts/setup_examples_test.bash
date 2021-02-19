#!/usr/bin/env bash

for f in ../../_examples/*; do
    if [ -d "$f" ]; then
        # Will not run if no directories are available
        go mod init
        go get -u github.com/kataras/iris/v12@master
        go mod download
        go run .
    fi
done

# git update-index --chmod=+x ./.github/scripts/setup_examples_test.bash