#!/bin/bash
set -e

OUTPUT_DIR="ios-app/Lib"
LIBRARY_NAME="librouter"

mkdir -p "$OUTPUT_DIR"

# Resolve SDK path dynamically via xcrun
SDK_PATH="$(xcrun --sdk iphoneos --show-sdk-path 2>/dev/null || true)"
if [ -z "$SDK_PATH" ] || [ ! -d "$SDK_PATH" ]; then
    SDK_PATH="/Applications/Xcode.app/Contents/Developer/Platforms/iPhoneOS.platform/Developer/SDKs/iPhoneOS.sdk"
fi

CLANG="$(xcrun --sdk iphoneos --find clang 2>/dev/null || true)"
if [ -z "$CLANG" ] || [ ! -f "$CLANG" ]; then
    CLANG="/Applications/Xcode.app/Contents/Developer/Toolchains/XcodeDefault.xctoolchain/usr/bin/clang"
fi

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
