#!/usr/bin/env bash

set -euo pipefail

# Uninstall script for GoFetch
# Removes installed binaries, symlinks, config and PATH lines added by install.sh

# Try to avoid getcwd errors when the caller's current directory was removed
cd "$(dirname "${BASH_SOURCE[0]}")" 2>/dev/null || cd / 2>/dev/null || true

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
# The installer wrote a block like:
#   # GoFetch
#   export PATH="${GOBIN}:$PATH"
# We'll remove a '# GoFetch' line followed by an export PATH line that contains the $GOBIN prefix.

SHELL_NAME="${SHELL##*/}"
case "$SHELL_NAME" in
  bash) SHELL_CONFIG="$HOME/.bashrc" ;;
  zsh) SHELL_CONFIG="$HOME/.zshrc" ;;
  *) SHELL_CONFIG="$HOME/.profile" ;;
esac

echo "Controllo e rimozione della riga PATH in: $SHELL_CONFIG"
if [[ -e "$SHELL_CONFIG" && -w "$SHELL_CONFIG" ]]; then
  tmpfile="$SHELL_CONFIG.gofetch.tmp"
  # Read line-by-line and skip the pattern block when matched
  {
    while IFS= read -r line || [[ -n "$line" ]]; do
      if [[ "$line" == "# GoFetch" ]]; then
        # read the next line (may be empty)
        if IFS= read -r nextline; then
          if [[ "$nextline" == export\ PATH=\"${GOBIN}:*" ]]; then
            # skip both lines
            continue
          else
            printf '%s\n' "$line"
            printf '%s\n' "$nextline"
          fi
        else
          # '# GoFetch' was the last line: just print it (conservative)
          printf '%s\n' "$line"
        fi
      else
        printf '%s\n' "$line"
      fi
    done < "$SHELL_CONFIG"
  } > "$tmpfile" && mv "$tmpfile" "$SHELL_CONFIG" && echo "Riga PATH rimossa (se presente) in $SHELL_CONFIG" || echo "Impossibile modificare $SHELL_CONFIG"
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
echo "Note:
 - Se hai eseguito lo script con sudo, HOME sarà /root e lo script opererà sulla home di root. Per disinstallare l'installazione dell'utente corrente, esegui lo script senza sudo.
 - Se alcuni file non sono stati rimossi per permessi, rimuovili manualmente (es: sudo rm /usr/local/bin/gofetch)."

exit 0
