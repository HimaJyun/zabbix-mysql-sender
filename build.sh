#!/bin/bash
TARGET=(
    "windows" "amd64"
    "darwin" "amd64"
    "linux" "amd64"
    "linux" "arm"
    "linux" "arm64"
)

mkdir -p ./.bin

for ((i=0; i < "${#TARGET[@]}"; i+=2)); do
    OS="${TARGET[$i]}"
    ARCH="${TARGET[$((i+1))]}"
    echo "OS=${OS} ARCH=${ARCH}"
    GOOS="${OS}" GOARCH="${ARCH}" go build -o "./.bin/zabbix-mysql-sender_${OS}_${ARCH}"
done
