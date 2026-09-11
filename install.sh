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
  # A Go installation's GOPATH/bin is not always in PATH. Use the
  # conventional per-user bin directory so the installer can configure it.
  GOBIN="${HOME}/.local/bin"
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
    case "${SHELL##*/}" in
      bash) SHELL_CONFIG="${HOME}/.bashrc" ;;
      zsh) SHELL_CONFIG="${HOME}/.zshrc" ;;
      *) SHELL_CONFIG="${HOME}/.profile" ;;
    esac
    PATH_LINE="export PATH=\"${GOBIN}:\$PATH\""
    touch "$SHELL_CONFIG"
    if ! grep -Fqx "$PATH_LINE" "$SHELL_CONFIG"; then
      printf '\n# GoFetch\n%s\n' "$PATH_LINE" >> "$SHELL_CONFIG"
      echo "PATH aggiornato in: $SHELL_CONFIG"
    fi
    echo
    echo "Per abilitarlo subito nella shell corrente esegui:"
    echo "  export PATH=\"$GOBIN:\$PATH\""
    ;;
esac

echo
echo "Avvia con: gofetch"