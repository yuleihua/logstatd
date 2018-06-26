#!/bin/bash


dos=`uname -s`
app=$1

for GOOS in $dos; do
    for GOARCH in 386 amd64; do
        go build -v -o ./dist/$app-$GOOS-$GOARCH
    done
done
