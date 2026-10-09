#!/bin/sh
# Builds the macOS Vision OCR helper and puts it inside the app bundle, then
# re-applies the ad-hoc signature so the bundle stays valid.
#
#   wails build ... && scripts/build-ocr-helper.sh [path/to/jumpstart.app]
#
# Without the helper, Settings > Search offers Tesseract or Ollama instead.
set -eu
cd "$(dirname "$0")/.."
APP="${1:-build/bin/jumpstart.app}"
[ -d "$APP/Contents/MacOS" ] || { echo "no app bundle at $APP" >&2; exit 1; }
OUT="$APP/Contents/MacOS/jumpstart-ocr"
# Same architecture as the app binary (wails builds for the host by default).
EXE="$(/usr/libexec/PlistBuddy -c 'Print :CFBundleExecutable' "$APP/Contents/Info.plist")"
ARCH="$(lipo -archs "$APP/Contents/MacOS/$EXE" | awk '{print $1}')"
xcrun swiftc -O -target "$ARCH-apple-macos12" -o "$OUT" tools/ocr-vision/main.swift
codesign --force --sign - "$OUT"
codesign --force --sign - "$APP"
echo "OCR helper: $OUT"
