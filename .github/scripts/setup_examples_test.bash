#!/usr/bin/env bash

for f in *; do
    if [ -d "$f" ]; then
        # Will not run if no directories are available
        go mod init
        go mod download
        go run .
    fi
done

# git update-index --chmod=+x ./.github/scripts/setup_examples_test.bash