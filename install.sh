#!/usr/bin/env bash
# GoFetch installer for macOS and Linux.
#
# Builds the sources in this directory (no network) and installs a stable
# `gofetch` binary into a per-user bin directory that is added to PATH.
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

BIN_DIR="${GOFETCH_BIN_DIR:-$HOME/.local/bin}"
mkdir -p "$BIN_DIR"

LATEST_COMMIT="$(git -C "$ROOT_DIR" log -1 --format='%h' 2>/dev/null || echo 'locale')"
LATEST_DATE="$(git -C "$ROOT_DIR" log -1 --format='%cI' 2>/dev/null || echo 'non disponibile')"
echo "Installazione del commit $LATEST_COMMIT ($LATEST_DATE) in: $BIN_DIR"

# Build to a temporary file first so a failed build never replaces a working
# installation. The binary is installed under the stable name `gofetch`, which
# is what users and the docs expect.
TEMP_BINARY="$(mktemp)"
cleanup() { rm -f "$TEMP_BINARY"; }
trap cleanup EXIT

(
  cd "$ROOT_DIR"
  go build -trimpath -ldflags="-s -w" -o "$TEMP_BINARY" .
)

# Refuse to clobber a file we did not create.
if [[ -e "$BIN_DIR/gofetch" && ! -L "$BIN_DIR/gofetch" && ! -f "$BIN_DIR/gofetch" ]]; then
  echo "Errore: $BIN_DIR/gofetch esiste e non è un file regolare." >&2
  exit 1
fi

mv -f "$TEMP_BINARY" "$BIN_DIR/gofetch"
chmod +x "$BIN_DIR/gofetch"
echo "GoFetch installato: $BIN_DIR/gofetch"

# Best-effort cleanup of legacy timestamped binaries from older installers.
shopt -s nullglob
for old_binary in "$BIN_DIR"/gofetch_*; do
  rm -f "$old_binary" && echo "Vecchia versione rimossa: $old_binary"
done
shopt -u nullglob

case ":${PATH}:" in
  *":${BIN_DIR}:"*) ;;
  *)
    case "${SHELL##*/}" in
      bash) SHELL_CONFIG="${HOME}/.bashrc" ;;
      zsh)  SHELL_CONFIG="${HOME}/.zshrc" ;;
      fish) SHELL_CONFIG="${HOME}/.config/fish/config.fish" ;;
      *)    SHELL_CONFIG="${HOME}/.profile" ;;
    esac
    if [[ "${SHELL##*/}" == "fish" ]]; then
      PATH_LINE="set -gx PATH \"${BIN_DIR}\" \$PATH"
    else
      PATH_LINE="export PATH=\"${BIN_DIR}:\$PATH\""
    fi
    mkdir -p "$(dirname "$SHELL_CONFIG")" 2>/dev/null || true
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
    if [[ "${SHELL##*/}" == "fish" ]]; then
      echo "  set -gx PATH \"$BIN_DIR\" \$PATH"
    else
      echo "  export PATH=\"$BIN_DIR:\$PATH\""
    fi
    ;;
esac

echo
echo "Avvia con: gofetch"
