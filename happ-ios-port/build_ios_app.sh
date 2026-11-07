#!/bin/bash
# Pipeline for building TrollStore-ready IPA:
# 1. Build Go core library (arm64, iOS c-archive)
# 2. Sync library and headers to ios-app/Lib
# 3. Generate Xcode project via xcodegen
# 4. Build HappPort & HappPortTunnel for iPhoneOS
# 5. Bundle with embedded hysteria binary, apply pseudo-signatures/entitlements, zip to .ipa
set -e

ROOT="$(cd "$(dirname "$0")" && pwd)"
APP_DIR="$ROOT/ios-app"
BUILD_DIR="$APP_DIR/build"

echo "==> [1/5] Building Go static library (arm64, iOS)"
"$ROOT/build_ios.sh"

echo "==> [2/5] Generating Xcode project"
cd "$APP_DIR"
xcodegen generate

echo "==> [3/5] Building Release binaries"
rm -rf "$BUILD_DIR"
mkdir -p "$BUILD_DIR"

xcodebuild -project HappPort.xcodeproj -scheme HappPort -configuration Release \
  -destination 'generic/platform=iOS' \
  -derivedDataPath "$BUILD_DIR/DerivedData" \
  CODE_SIGNING_ALLOWED=NO \
  CODE_SIGN_IDENTITY="" \
  CODE_SIGNING_REQUIRED=NO \
  clean build

echo "==> [4/5] Packaging Payload for TrollStore"
PAYLOAD_DIR="$BUILD_DIR/Payload"
mkdir -p "$PAYLOAD_DIR"

APP_PATH=$(find "$BUILD_DIR/DerivedData/Build/Products/Release-iphoneos" -name "HappPort.app" -type d | head -n 1)
if [ -z "$APP_PATH" ]; then
  echo "Error: HappPort.app not found in build products"
  exit 1
fi

cp -r "$APP_PATH" "$PAYLOAD_DIR/"

# Ensure hysteria darwin/arm64 binary is available
if [ ! -f "$ROOT/bin/hysteria" ]; then
  echo "Fetching hysteria-darwin-arm64..."
  mkdir -p "$ROOT/bin"
  curl -sLo "$ROOT/bin/hysteria" https://github.com/apernet/hysteria/releases/latest/download/hysteria-darwin-arm64 || true
fi

if [ -f "$ROOT/bin/hysteria" ]; then
  echo "Embedding hysteria binary..."
  cp "$ROOT/bin/hysteria" "$PAYLOAD_DIR/HappPort.app/hysteria"
  chmod +x "$PAYLOAD_DIR/HappPort.app/hysteria"
fi

# Fake-sign with entitlements using ldid if available
if command -v ldid &> /dev/null; then
  echo "Applying entitlements via ldid..."
  ldid -S"$APP_DIR/HappPort/HappPort.entitlements" "$PAYLOAD_DIR/HappPort.app/HappPort"
  if [ -d "$PAYLOAD_DIR/HappPort.app/PlugIns/HappPortTunnel.appex" ]; then
    ldid -S"$APP_DIR/HappPortTunnel/HappPortTunnel.entitlements" "$PAYLOAD_DIR/HappPort.app/PlugIns/HappPortTunnel.appex/HappPortTunnel"
  fi
fi

echo "==> [5/5] Creating IPA archive"
cd "$BUILD_DIR"
zip -r9 "$ROOT/HappPort-TrollStore.ipa" Payload

echo ""
echo "=== SUCCESS ==="
echo "IPA created at: $ROOT/HappPort-TrollStore.ipa"
