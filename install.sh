#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REMOTE_REPO="${GOFETCH_REPO:-}"
REMOTE_REF="${GOFETCH_REF:-}"

if [[ -z "$REMOTE_REPO" ]] && git -C "$ROOT_DIR" rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  REMOTE_REPO="$(git -C "$ROOT_DIR" remote get-url origin 2>/dev/null || true)"
fi
if [[ -z "$REMOTE_REPO" ]]; then
  REMOTE_REPO="git@github.com:ciao22king/GoFetch.git"
fi
if [[ -z "$REMOTE_REF" ]] && git -C "$ROOT_DIR" rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  REMOTE_REF="$(git -C "$ROOT_DIR" symbolic-ref --short refs/remotes/origin/HEAD 2>/dev/null || true)"
  REMOTE_REF="${REMOTE_REF#origin/}"
fi
if [[ -z "$REMOTE_REF" ]]; then
  REMOTE_REF="main"
fi

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

SOURCE_DIR="$ROOT_DIR"
TEMP_DIR=""
cleanup() {
  if [[ -n "$TEMP_DIR" ]]; then
    rm -rf "$TEMP_DIR"
  fi
}
trap cleanup EXIT

if [[ "${GOFETCH_LOCAL:-0}" != "1" ]]; then
  TEMP_DIR="$(mktemp -d)"
  SOURCE_DIR="$TEMP_DIR/GoFetch"
  echo "Recupero l'ultimo commit da $REMOTE_REPO ($REMOTE_REF)..."
  if ! git clone --depth 1 --branch "$REMOTE_REF" "$REMOTE_REPO" "$SOURCE_DIR"; then
    echo "Errore: non riesco a recuperare l'ultima copia dal repository remoto." >&2
    echo "Se vuoi compilare i file già presenti localmente, usa: GOFETCH_LOCAL=1 bash install.sh" >&2
    exit 1
  fi
fi

LATEST_COMMIT="$(git -C "$SOURCE_DIR" log -1 --format='%h' 2>/dev/null || echo 'locale')"
LATEST_DATE="$(git -C "$SOURCE_DIR" log -1 --format='%cI' 2>/dev/null || echo 'non disponibile')"

echo "Installazione del commit $LATEST_COMMIT ($LATEST_DATE) in: $GOBIN"
(
  cd "$SOURCE_DIR"
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
    if [[ ! -e "$SHELL_CONFIG" ]]; then
      touch "$SHELL_CONFIG" 2>/dev/null || true
    fi
    if [[ -w "$SHELL_CONFIG" ]]; then
      if ! grep -Fqx "$PATH_LINE" "$SHELL_CONFIG"; then
        printf '\n# GoFetch\n%s\n' "$PATH_LINE" >> "$SHELL_CONFIG"
        echo "PATH aggiornato in: $SHELL_CONFIG"
      fi
    else
      echo "Nota: non posso modificare $SHELL_CONFIG; aggiungi manualmente GoFetch al PATH."
    fi
    echo
    echo "Per abilitarlo subito nella shell corrente esegui:"
    echo "  export PATH=\"$GOBIN:\$PATH\""
    ;;
esac

echo
echo "Avvia con: gofetch"