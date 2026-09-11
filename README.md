# GoFetch

GoFetch is a small, focused terminal UI for cloning GitHub and GitLab repositories without remembering the exact `git clone` incantation.

![GoFetch terminal UI](https://placehold.co/1200x700/0f172a/e2e8f0?text=GoFetch)

## Why

The normal flow is simple, but it is still full of tiny decisions: HTTPS or SSH, where to put the folder, and whether the URL is even valid. GoFetch turns that into a calm flow and remembers every successful fetch:

1. Choose a previous fetch with `↑`/`↓` and press `Enter` to open its folder, or press `n` for a new clone.
2. Paste a GitHub or GitLab URL.
3. Choose the parent directory.
4. GoFetch runs `git clone`, attempts a project build, and saves the result in its local history.

It supports:

- GitHub and GitLab HTTPS URLs
- GitHub and GitLab SSH URLs
- GitLab nested groups
- `~` in destination paths
- Friendly validation and clone errors
- A keyboard-first Bubble Tea interface
- Persistent fetch history in `~/.config/gofetch/history.json`
- One-key opening of previously fetched repository folders
- Automatic build detection for Go, Rust, and Node projects

After a successful clone, GoFetch runs `go build ./...`, `cargo build`, or `npm run build` when the corresponding project manifest is present. If no supported build setup is found, the clone still completes normally. Node build scripts and Rust build scripts can execute project-defined code, so use automatic builds only for repositories you trust.

## Install

Requirements:

- Go 1.23+
- Git available on your `PATH`

```bash
go install github.com/ciao22king/GoFetch@latest
```

Or build locally:

```bash
go build -o gofetch .
./gofetch
```

Installazione automatica:

```bash
chmod +x install.sh
./install.sh
```

Lo script compila esclusivamente i file presenti nella directory locale del progetto. Non usa la rete e non usa il numero di versione per scegliere cosa installare. Mostra la data dell’ultimo commit locale solo come informazione. Poi installa GoFetch in `~/.local/bin`, aggiungendo automaticamente la directory al file di configurazione della shell. Dopo l’installazione apri un nuovo terminale oppure esegui il comando `export PATH="$HOME/.local/bin:$PATH"` mostrato dallo script.

Puoi cambiare destinazione con:

```bash
GOFETCH_BIN_DIR="$HOME/.local/bin" ./install.sh
```

## Usage

Interactive mode:

```bash
gofetch
```

Start with a URL:

```bash
gofetch --url git@github.com:owner/project.git
```

Choose a parent directory:

```bash
gofetch --dir ~/Code
```

## Keyboard shortcuts

| Key | Action |
| --- | --- |
| `Enter` | Continue / clone |
| `↑` / `↓` | Select a previous fetch |
| `n` | Start a new fetch |
| `Ctrl+V` / `Cmd+V` | Incolla URL o percorso dalla clipboard |
| `Esc` | Go back |
| `q` | Quit |

## Roadmap

- Clone destination preview
- Optional branch selection
- GitHub/GitLab API search
- Clone progress stream inside the TUI