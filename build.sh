#!/bin/bash
set -e
function gvt {
	if [ ! -d "vendor/$2" ]; then
		/usr/bin/env gvt $1 $2
	fi
}
# function has precedence over binary
gvt fetch github.com/elazarl/go-bindata-assetfs
gvt fetch github.com/gorilla/websocket
gvt fetch github.com/paulstuart/ping

go generate
GOOSES=${1:-"darwin linux windows"}
GOARCHS=${2:-"amd64 386"}
for GOOS in $GOOSES; do
	for GOARCH in $GOARCHS; do
		echo "Building $GOOS/$GOARCH"
		if [ "$GOOS" == "windows" ]; then
			docker run --rm -e CGO_ENABLED=0 -e GOOS=$GOOS -e GOARCH=$GOARCH -v `pwd`:/go/src/app -w /go/src/app golang:1.9-alpine go build -o build/isiton-$GOOS-$GOARCH.exe .
		else
			docker run --rm -e CGO_ENABLED=0 -e GOOS=$GOOS -e GOARCH=$GOARCH -v `pwd`:/go/src/app -w /go/src/app golang:1.9-alpine go build -o build/isiton-$GOOS-$GOARCH .
		fi
	done
done