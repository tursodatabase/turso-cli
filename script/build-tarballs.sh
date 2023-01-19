#!/bin/bash

version=0.0.0

build_tarball () {
  os=$1
  arch=$2
  program=$3
  target=$os-$arch
  GOARCH=$arch GOOS=$os go build -o build/turso-$os-$arch/$program main.go
  tar -C build -czvf "turso-$version-$target.tar.gz" "turso-$target/"
}

build_tarball darwin amd64 turso
build_tarball darwin arm64 turso
build_tarball linux amd64 turso
build_tarball windows amd64 turso.exe
