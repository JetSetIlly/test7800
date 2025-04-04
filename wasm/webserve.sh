#/bin/bash

env GOOS=js GOARCH=wasm go build -pgo=auto -gcflags '-c 3 -B -wb=false -l -l -l -l' -o www/test7800.wasm .
go run ./httpd
