#!/bin/bash
# Build the magpie fnOS .fpk. Requires fnpack (https://developer.fnnas.com/docs/cli/fnpack/).
set -e
HERE=$(cd "$(dirname "$0")" && pwd); cd "$HERE"
VERSION=0.1.4289
BIN_URL="https://github.com/yetone/magpie-releases/releases/download/v${VERSION}/magpie-cli-linux-amd64"
mkdir -p app/bin
if [ ! -s app/bin/magpie ]; then echo "==> downloading magpie ${VERSION}"; curl -fsSL -o app/bin/magpie "$BIN_URL"; fi
chmod 755 app/bin/magpie
if [ ! -s app/bin/magpie-proxy ]; then
  GO=${GO:-go}
  command -v "$GO" >/dev/null 2>&1 || { echo "app/bin/magpie-proxy missing and Go not found"; exit 1; }
  echo "==> building gateway proxy"
  ( cd proxy && CGO_ENABLED=0 "$GO" build -trimpath -ldflags "-s -w" -o ../app/bin/magpie-proxy . )
fi
chmod 755 app/bin/magpie-proxy
FNPACK=${FNPACK:-fnpack}
command -v "$FNPACK" >/dev/null 2>&1 || { echo "fnpack not found; install from https://developer.fnnas.com/docs/cli/fnpack/"; exit 1; }
"$FNPACK" build --directory "$HERE"
echo "==> built: $HERE/magpie.fpk"
