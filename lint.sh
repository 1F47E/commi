#!/bin/bash

set -e

echo "Running gofmt..."
gofmt_output=$(gofmt -l -s -w .)
if [ -n "$gofmt_output" ]; then
    echo "gofmt made changes to the following files:"
    echo "$gofmt_output"
else
    echo "gofmt made no changes."
fi

echo "Running go vet..."
go vet ./...

echo "Running golint..."
golint -set_exit_status ./...

echo "Running staticcheck..."
staticcheck ./...

echo "All linters passed successfully!"
