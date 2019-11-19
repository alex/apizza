#!/usr/bin/bash

if [ "$TRAVIS_GO_VERSION" = "1.11" ] || [ "$GO111MODULE" != on ]; then
	export GO111MODULE=off
    go list -f '{{ join .Imports "\n" }}' ./... | \
		grep -P '^(github.com|gopkg.in)/.*' | \
		grep -v "`go list`" | \
		awk '{print}' ORS=' ' | \
		go get -u
fi

go install