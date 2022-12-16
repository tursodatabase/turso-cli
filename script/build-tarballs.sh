#!/bin/bash

version=0.0.0

build_tarball () {
  os=$1
  arch=$2
  program=$3
  target=$os-$arch
  GOARCH=$arch GOOS=$os go build -o build/ikuctl-$os-$arch/$program main.go
  tar -C build -czvf "ikuctl-$version-$target.tar.gz" "ikuctl-$target/"
}

build_tarball darwin amd64 ikuctl
build_tarball darwin arm64 ikuctl
build_tarball linux amd64 ikuctl
build_tarball windows amd64 ikuctl.exe
