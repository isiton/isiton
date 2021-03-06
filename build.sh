#!/bin/bash
set -e
NAMES=$(find cmd/* -type d | xargs -n1 basename)
if [ ! -z "$1" ]; then
	NAMES="$1"
fi

go-bindata -o assets/bindata.go -pkg assets -nomemcopy public_html/...

for NAME in $NAMES; do
	OSES=${OSS:-"windows linux"}
	ARCHS=${ARCHS:-"amd64"}
	for ARCH in $ARCHS; do
		for OS in $OSES; do
			echo $OS $ARCH $NAME
			GOOS=${OS} GOARCH=${ARCH} CGO_ENABLED=0 GOARM=7 go build -o build/${NAME}-${OS}-${ARCH} cmd/${NAME}/*.go
			if [ $? -eq 0 ]; then
				echo OK
			fi
			if [ "$OS" == "windows" ]; then
				mv build/${NAME}-${OS}-${ARCH} build/${NAME}-${OS}-${ARCH}.exe
			fi
		done
	done
done
