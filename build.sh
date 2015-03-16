#!/bin/bash

# Create the GOFILES variable with a list of all .go files
# in the directory.
GOFILES="$(ls *.go)"

# Build the Go application using the GOFILES variable
go build -o bin/goweather $GOFILES
