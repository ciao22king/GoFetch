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

SOURCE_DIR="$ROOT_DIR"
LATEST_COMMIT="$(git -C "$SOURCE_DIR" log -1 --format='%h' 2>/dev/null || echo 'locale')"
LATEST_DATE="$(git -C "$SOURCE_DIR" log -1 --format='%cI' 2>/dev/null || echo 'non disponibile')"

# Converti la data nel formato YYYY-MM-DD_HH-MM-SS per il nome del file
BINARY_NAME="gofetch_${LATEST_DATE}"
# Rimuovi caratteri problematici (: e +)
BINARY_NAME="$(echo "$BINARY_NAME" | sed 's/[+:]//g' | sed 's/Z$//')"

TEMP_BINARY="$(mktemp)"
cleanup() {
  if [[ -f "$TEMP_BINARY" ]]; then
    rm -f "$TEMP_BINARY"
  fi
}
trap cleanup EXIT

echo "Installazione del commit $LATEST_COMMIT ($LATEST_DATE) in: $GOBIN"
(
  cd "$SOURCE_DIR"
  go build -trimpath -ldflags="-s -w" -o "$TEMP_BINARY" .
)

# Copia il binario con il nome datato
mv -f "$TEMP_BINARY" "$GOBIN/$BINARY_NAME"
chmod +x "$GOBIN/$BINARY_NAME"
echo "GoFetch installato: $GOBIN/$BINARY_NAME"

# Crea un symlink "gofetch" che punta all'ultima versione
ln -sf "$BINARY_NAME" "$GOBIN/gofetch"
echo "Symlink creato: $GOBIN/gofetch -> $BINARY_NAME"

# Rimuovi le vecchie versioni datate (mantieni solo l'ultima)
echo "Rimozione delle vecchie versioni..."
(
  cd "$GOBIN"
  # Trova tutti i binari gofetch_* ordinati in reverse, salta il primo (l'ultimo installato)
  ls -1 gofetch_* 2>/dev/null | sort -r | tail -n +2 | while read old_binary; do
    if rm -f "$old_binary" 2>/dev/null; then
      echo "Vecchia versione rimossa: $old_binary"
    else
      echo "Nota: non posso rimuovere: $old_binary"
    fi
  done
)

# Rimuovi eventuali altre installazioni in percorsi diversi
declare -a OLD_BINARIES=("$HOME/.local/bin/gofetch" "/usr/local/bin/gofetch")
GO_GOBIN="$(go env GOBIN)"
if [[ -n "$GO_GOBIN" ]]; then
  OLD_BINARIES+=("$GO_GOBIN/gofetch")
fi
IFS=: read -ra GOPATH_ENTRIES <<< "$(go env GOPATH)"
for go_path in "${GOPATH_ENTRIES[@]}"; do
  OLD_BINARIES+=("$go_path/bin/gofetch")
done

declare -A SEEN_BINARIES=()
for old_binary in "${OLD_BINARIES[@]}"; do
  [[ -n "$old_binary" ]] || continue
  [[ -n "${SEEN_BINARIES[$old_binary]:-}" ]] && continue
  SEEN_BINARIES["$old_binary"]=1
  [[ "$old_binary" == "$GOBIN/gofetch" ]] && continue
  if [[ -f "$old_binary" || -L "$old_binary" ]]; then
    if rm -f "$old_binary" 2>/dev/null; then
      echo "Vecchia installazione rimossa: $old_binary"
    else
      echo "Nota: non posso rimuovere la vecchia installazione: $old_binary"
    fi
  fi
done

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
