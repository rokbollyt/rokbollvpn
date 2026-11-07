#!/bin/bash
set -e

OUTPUT_DIR="ios-app/Lib"
LIBRARY_NAME="librouter"

mkdir -p "$OUTPUT_DIR"

echo "==> Resolving Go dependencies..."
cd go-core
go mod tidy
cd ..

# Resolve SDK path dynamically via xcrun
SDK_PATH="$(xcrun --sdk iphoneos --show-sdk-path 2>/dev/null || true)"
if [ -z "$SDK_PATH" ] || [ ! -d "$SDK_PATH" ]; then
    DEVELOPER_DIR="$(xcode-select -p 2>/dev/null || echo '/Applications/Xcode.app/Contents/Developer')"
    SDK_PATH="$DEVELOPER_DIR/Platforms/iPhoneOS.platform/Developer/SDKs/iPhoneOS.sdk"
fi

CLANG="$(xcrun --find clang 2>/dev/null || which clang || echo "clang")"

echo "==> Using SDK: $SDK_PATH"
echo "==> Using Clang: $CLANG"

export GOARCH=arm64
export GOOS=ios
export CGO_ENABLED=1
export SDK_PATH="$SDK_PATH"
export CC="$CLANG -isysroot $SDK_PATH -arch arm64 -miphoneos-version-min=15.0"
export CXX="${CLANG}++ -isysroot $SDK_PATH -arch arm64 -miphoneos-version-min=15.0"
export CGO_CFLAGS="-isysroot $SDK_PATH -arch arm64 -miphoneos-version-min=15.0"
export CGO_LDFLAGS="-isysroot $SDK_PATH -arch arm64 -miphoneos-version-min=15.0"

echo "==> Building Go core for iOS (arm64)..."

cd go-core
go build \
    -v \
    -buildmode=c-archive \
    -ldflags="-w -s" \
    -trimpath \
    -o "../$OUTPUT_DIR/$LIBRARY_NAME.a" \
    .

cd ..
echo "Build complete: $OUTPUT_DIR/$LIBRARY_NAME.a and $OUTPUT_DIR/$LIBRARY_NAME.h"
