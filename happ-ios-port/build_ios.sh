#!/bin/bash
set -e

OUTPUT_DIR="ios-app/Lib"
LIBRARY_NAME="librouter"

XCODE_PATH="${XCODE_PATH:-/Applications/Xcode.app}"
DEVELOPER_DIR="$XCODE_PATH/Contents/Developer"
SDK_PATH="$DEVELOPER_DIR/Platforms/iPhoneOS.platform/Developer/SDKs/iPhoneOS.sdk"
CLANG="$DEVELOPER_DIR/Toolchains/XcodeDefault.xctoolchain/usr/bin/clang"

mkdir -p "$OUTPUT_DIR"

if [ ! -d "$SDK_PATH" ]; then
    echo "SDK not found: $SDK_PATH"
    exit 1
fi

if [ ! -f "$CLANG" ]; then
    echo "Compiler not found: $CLANG"
    exit 1
fi

export GOARCH=arm64
export GOOS=ios
export CGO_ENABLED=1
export SDK_PATH="$SDK_PATH"
export CC="$CLANG -isysroot $SDK_PATH -arch arm64 -miphoneos-version-min=14.0"
export CXX="${CLANG}++ -isysroot $SDK_PATH -arch arm64 -miphoneos-version-min=14.0"
export CGO_CFLAGS="-isysroot $SDK_PATH -arch arm64 -miphoneos-version-min=14.0"
export CGO_LDFLAGS="-isysroot $SDK_PATH -arch arm64 -miphoneos-version-min=14.0"

echo "Building Go core for iOS (arm64)..."

cd go-core
go build \
    -buildmode=c-archive \
    -ldflags="-w -s" \
    -trimpath \
    -o "../$OUTPUT_DIR/$LIBRARY_NAME.a" \
    .

cd ..
echo "Build complete: $OUTPUT_DIR/$LIBRARY_NAME.a and $OUTPUT_DIR/$LIBRARY_NAME.h"

