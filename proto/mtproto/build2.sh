#!/bin/sh

SRC_DIR=.
GOGOPROTO_PATH=$(go env GOPATH)/pkg/mod/github.com/gogo/protobuf@v1.3.2/protobuf

# Format all .proto files
if command -v clang-format >/dev/null 2>&1; then
  find . -name '*.proto' -exec clang-format -i {} \;
fi

# NOTE: Only regenerate cityactivity.proto. The other .pb.go files were generated
# with a different version of protobuf/mtprotoc and have different naming conventions.
# Regenerating them would break codec_schema.tl.pb.go compatibility.
#
# To regenerate ALL proto files, use the original proto_sources/build2.sh with the
# correct GOPATH/src directory structure. DO NOT run this script with *.proto.

protoc -I=$SRC_DIR --proto_path=$GOGOPROTO_PATH:./ \
    --gogo_out=plugins=grpc,paths=source_relative,Mgoogle/protobuf/wrappers.proto=github.com/gogo/protobuf/types:. \
    $SRC_DIR/schema.tl.cityactivity.proto

# Fix enum naming convention to match existing codebase
if [ -f schema.tl.cityactivity.pb.go ]; then
  sed -i '' 's/TLConstructor_CRC32_UNKNOWN/CRC32_UNKNOWN/g' schema.tl.cityactivity.pb.go 2>/dev/null || \
  sed -i 's/TLConstructor_CRC32_UNKNOWN/CRC32_UNKNOWN/g' schema.tl.cityactivity.pb.go
fi

gofmt -w schema.tl.cityactivity.pb.go
