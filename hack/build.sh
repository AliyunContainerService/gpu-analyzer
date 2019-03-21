#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

GIT_SHA=`git rev-parse HEAD || echo "GitNotFound"`
gitHash="github.com/AliyunContainerService/gpu-analyzer/app/version.GitSHA=${GIT_SHA}"

go_ldflags="-X ${gitHash}"

echo "building ${GIT_SHA}..."
go build -ldflags "$go_ldflags" -o gpu-analyzer ./app/gpu-analyzer/main.go

echo "build finished"
