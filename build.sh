#!/bin/bash
set -e

echo "=== Building libqqcliput.dylib ==="
clang -shared -fobjc-arc \
    -install_name @executable_path/libqqcliput.dylib \
    -framework Foundation \
    -framework CoreGraphics \
    -framework CoreImage \
    -framework Vision \
    -o libqqcliput.dylib qqcliput.m

echo "=== Building Go binary ==="
go build -buildmode=pie -o qqcliput .

echo "=== Creating .app bundle ==="
mkdir -p qqcliput.app/Contents/MacOS
mkdir -p qqcliput.app/Contents/Resources
cp qqcliput qqcliput.app/Contents/MacOS/
cp Info.plist qqcliput.app/Contents/
cp libqqcliput.dylib qqcliput.app/Contents/MacOS/

echo "=== Done ==="
echo ""
echo "Run: ./qqcliput.app/Contents/MacOS/qqcliput"
echo "Or copy qqcliput.app to /Applications"
