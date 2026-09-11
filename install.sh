#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

if ! command -v go >/dev/null 2>&1; then
  echo "Errore: Go non è installato o non è nel PATH." >&2
  echo "Installa Go 1.23+ da https://go.dev/dl/ e riprova." >&2
  exit 1
fi

if ! command -v git >/dev/null 2>&1; then
  echo "Errore: Git non è installato o non è nel PATH." >&2
  exit 1
fi

if [[ -n "${GOFETCH_BIN_DIR:-}" ]]; then
  GOBIN="$GOFETCH_BIN_DIR"
else
  GOBIN="$(go env GOBIN)"
  if [[ -z "$GOBIN" ]]; then
    GOBIN="$(go env GOPATH)/bin"
  fi
fi

mkdir -p "$GOBIN"

echo "Installazione di GoFetch in: $GOBIN"
(
  cd "$ROOT_DIR"
  go build -trimpath -ldflags="-s -w" -o "$GOBIN/gofetch" .
)

chmod +x "$GOBIN/gofetch"
echo "GoFetch installato: $GOBIN/gofetch"

case ":${PATH}:" in
  *":${GOBIN}:"*) ;;
  *)
    echo
    echo "Nota: aggiungi questa directory al PATH per eseguire 'gofetch' ovunque:"
    echo "  export PATH=\"\$PATH:$GOBIN\""
    ;;
esac

echo
echo "Avvia con: gofetch"