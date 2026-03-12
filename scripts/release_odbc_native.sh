#!/usr/bin/env bash
set -euo pipefail

TAG="${GITHUB_REF_NAME:?GITHUB_REF_NAME is required}"
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

case "$ARCH" in
  x86_64) GOARCH="amd64" ;;
  arm64|aarch64) GOARCH="arm64" ;;
  *) echo "Unsupported architecture: $ARCH" >&2; exit 1 ;;
esac

mkdir -p dist

LDFLAGS="-s -w -X oslo/cmd.Version=${TAG#v} -X oslo/cmd.BuildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ) -X oslo/cmd.GitCommit=$(git rev-parse --short HEAD) -X oslo/cmd.Edition=odbc-full"

if [[ "$OS" == "darwin" ]]; then
  ODBC_PREFIX="$(brew --prefix unixodbc)"
  export CGO_ENABLED=1
  export CGO_CFLAGS="-I${ODBC_PREFIX}/include"
  export CGO_LDFLAGS="-L${ODBC_PREFIX}/lib"
else
  export CGO_ENABLED=1
  export CGO_LDFLAGS="-lodbc"
fi

BIN="connect-dbms"
OUT="dist/connect-dbms-odbc-${TAG}-${OS}-${GOARCH}"

GOOS="$OS" GOARCH="$GOARCH" go build -tags "tibero cubrid" -ldflags "$LDFLAGS" -o "$BIN" .
tar -czf "${OUT}.tar.gz" "$BIN"

gh release upload "$TAG" "${OUT}.tar.gz" --clobber
