// +build tools

package internal

// This file holds references to the various tools needed at build-time so that
// go mod fetches them. This is the current best practice according to:
// https://github.com/golang/go/issues/25922

import (
	_ "github.com/ckaznocha/protoc-gen-lint"
	_ "github.com/golang/protobuf/protoc-gen-go"
	_ "github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway"
	_ "github.com/grpc-ecosystem/grpc-gateway/protoc-gen-swagger"
	_ "github.com/grpc-ecosystem/grpc-gateway/runtime"
)
