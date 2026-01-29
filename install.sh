#!/bin/bash

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "🔨 Building Go..."
cd "$SCRIPT_DIR/src"
./make.bash

echo ""
echo "✅ Build completed!"
echo ""
echo "Setup environment:"
echo "  export GOROOT=$SCRIPT_DIR"
echo "  export PATH=\$GOROOT/bin:\$PATH"
