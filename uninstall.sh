#!/usr/bin/env bash

set -euo pipefail

# Uninstall script for GoFetch
# Removes installed binaries, symlinks, config and PATH lines added by install.sh

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

if [[ -n "${GOFETCH_BIN_DIR:-}" ]]; then
  GOBIN="$GOFETCH_BIN_DIR"
else
  GOBIN="${HOME}/.local/bin"
fi

echo "Usando GOBIN: $GOBIN"

# Helper: try to remove a file or symlink, report result
remove_file() {
  local f="$1"
  if [[ -L "$f" || -f "$f" ]]; then
    if rm -f "$f" 2>/dev/null; then
      echo "Rimosso: $f"
    else
      echo "Impossibile rimuovere (permessi?): $f"
    fi
  fi
}

# 1) Remove dated binaries and the generic symlink in the chosen GOBIN
if [[ -d "$GOBIN" ]]; then
  echo "Rimuovo binari in $GOBIN ..."
  # remove gofetch (symlink or file)
  remove_file "$GOBIN/gofetch"
  # remove any gofetch_* dated binaries
  shopt -s nullglob 2>/dev/null || true
  for b in "$GOBIN"/gofetch_*; do
    remove_file "$b"
  done
else
  echo "Directory $GOBIN non esistente, salto la rimozione lì."
fi

# 2) Remove other common installation locations
echo "Controllo altre posizioni comuni..."
declare -a OLD_BINARIES=("$HOME/.local/bin/gofetch" "/usr/local/bin/gofetch" "/usr/bin/gofetch")
GO_GOBIN="$(command -v go >/dev/null 2>&1 && go env GOBIN || echo '')"
if [[ -n "$GO_GOBIN" ]]; then
  OLD_BINARIES+=("$GO_GOBIN/gofetch")
fi
if command -v go >/dev/null 2>&1; then
  IFS=: read -ra GOPATH_ENTRIES <<< "$(go env GOPATH)"
  for go_path in "${GOPATH_ENTRIES[@]}"; do
    [[ -n "$go_path" ]] || continue
    OLD_BINARIES+=("$go_path/bin/gofetch")
    OLD_BINARIES+=("$go_path/bin/gofetch_*")
  done
fi

# Deduplicate and attempt removal
declare -A SEEN=()
for candidate in "${OLD_BINARIES[@]}"; do
  [[ -n "$candidate" ]] || continue
  if [[ -n "${SEEN[$candidate]:-}" ]]; then
    continue
  fi
  SEEN[$candidate]=1
  # Expand globs safely
  if [[ "$candidate" == *"*" ]]; then
    for f in $candidate; do
      remove_file "$f"
    done
  else
    remove_file "$candidate"
  fi
done

# 3) Remove config and history
CONFIG_DIR="$HOME/.config/gofetch"
if [[ -d "$CONFIG_DIR" ]]; then
  if rm -rf "$CONFIG_DIR" 2>/dev/null; then
    echo "Directory di configurazione rimossa: $CONFIG_DIR"
  else
    echo "Impossibile rimuovere la directory di configurazione: $CONFIG_DIR (permessi?)"
  fi
else
  echo "Nessuna directory di configurazione trovata in $CONFIG_DIR"
fi

# 4) Remove PATH line added to shell config by install.sh
PATH_LINE='export PATH="${GOBIN}:$PATH"'
# We must match the literal line that was written during install; install.sh used
# PATH_LINE="export PATH=\"${GOBIN}:\$PATH\"" and a preceding comment '# GoFetch'

SHELL_NAME="${SHELL##*/}"
case "$SHELL_NAME" in
  bash) SHELL_CONFIG="$HOME/.bashrc" ;;
  zsh) SHELL_CONFIG="$HOME/.zshrc" ;;
  *) SHELL_CONFIG="$HOME/.profile" ;;
esac

echo "Controllo e rimozione della riga PATH in: $SHELL_CONFIG"
if [[ -e "$SHELL_CONFIG" && -w "$SHELL_CONFIG" ]]; then
  # Remove the block: a line with '# GoFetch' followed by the exact PATH_LINE
  # Use awk to safely remove occurrences
  awk -v gobin="$GOBIN" '
    BEGIN { p = 0 }
    {
      if ($0 == "# GoFetch") {
        getline nextline
        expected = "export PATH=\"" gobin ":$PATH\""
        # Note: when awk expands environment PATH it won't match, so compare using prefix
        if (nextline ~ /^export PATH=\"/ && nextline ~ gobin) {
          # skip both lines
          next
        } else {
          print "# GoFetch"
          print nextline
        }
      } else {
        print $0
      }
    }
  ' "$SHELL_CONFIG" > "$SHELL_CONFIG.gofetch.tmp" && mv "$SHELL_CONFIG.gofetch.tmp" "$SHELL_CONFIG" && echo "Riga PATH rimossa (se presente) in $SHELL_CONFIG" || echo "Impossibile modificare $SHELL_CONFIG"
else
  echo "Non posso modificare $SHELL_CONFIG (non esiste o non è scrivibile). Rimuovi manualmente le righe '# GoFetch' e la linea PATH se presenti."
fi

# 5) Report final status
echo
echo "Rimozione completata. Azioni riepilogo:"
echo " - Binari rimossi da: $GOBIN (se presenti)"
echo " - Altre posizioni controllate: /usr/local/bin, /usr/bin, GOPATH e GOBIN di Go"
echo " - Directory di configurazione rimossa: $CONFIG_DIR (se presente)"
echo " - Linee PATH rimosse dallo shell config: $SHELL_CONFIG (se possibile)"

echo
echo "Se qualche file non è stato rimosso per motivi di permessi, esegui lo script con i permessi appropriati o rimuovili manualmente con sudo (es: sudo rm /usr/local/bin/gofetch)."

exit 0
