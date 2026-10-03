# GoFetch

GoFetch is a small, focused terminal UI for cloning GitHub and GitLab repositories without remembering the exact `git clone` incantation. It runs on Windows, macOS and Linux.

![GoFetch terminal UI](https://placehold.co/1200x700/0f172a/e2e8f0?text=GoFetch)

## Why

The normal flow is simple, but it is still full of tiny decisions: HTTPS or SSH, where to put the folder, and whether the URL is even valid. GoFetch turns that into a calm flow and remembers every successful fetch:

1. Choose a previous fetch with `↑`/`↓` and press `Enter` to open its folder, press `u` to update it with a fast-forward `git pull`, or press `n` for a new clone.
2. Paste a GitHub or GitLab URL.
3. Optionally type a branch — press `Tab` to complete from the remote's actual branch list (leave it empty for the remote default branch).
4. Choose the parent directory (press `Tab` to cycle through recent folders), check the destination preview, and press `s` if you want a fast shallow clone.
5. GoFetch runs `git clone`, streams its progress live, attempts a project build, and saves the result in its local history.

It supports:

- GitHub and GitLab HTTPS, SSH and `ssh://` URLs
- GitLab nested groups
- Self-hosted / other Git forges (any real host is accepted as a generic remote)
- Local repositories via `file://` URLs
- `~` in destination paths, with an absolute clone target computed up front
- An optional branch selection, with `--branch` for non-interactive use and `Tab` completion from the remote's branch list in the TUI
- Shallow clones (`--depth N`, or `s` on the destination screen) for fast fetches of large repositories
- Updating an existing clone in place with a safe fast-forward `git pull` (`u` in the history screen)
- A non-interactive mode (`--yes`) for scripts and CI, printing the destination path to stdout
- A config file (`config.json` next to the history) with default directory, clone depth and build behavior
- A live clone progress stream inside the TUI
- A destination preview before the clone starts
- A safe overwrite confirmation when the destination folder already exists
- Cancellable clones (`Esc` while cloning)
- Friendly, actionable clone errors
- A keyboard-first Bubble Tea interface with live history search
- Persistent fetch history in `~/.config/gofetch/history.json` (`%AppData%\gofetch\history.json` on Windows)
- One-key opening and path-copying of previous fetch folders
- Automatic build detection for Go, Rust, Node (npm/yarn/pnpm), Java (Maven/Gradle), Make and Python

## Cross-platform

| Platform | Open folder | Clipboard | Installer |
| --- | --- | --- | --- |
| macOS | `open` | `pbcopy`/`pbpaste` | `install.sh` |
| Linux | `xdg-open` | `xclip`/`xsel`/`wl-copy` | `install.sh` |
| Windows | Explorer | native | `install.ps1` |

Repository names are sanitized so a clone is safe on every platform: characters Windows forbids (`\ / : * ? " < > |`), trailing dots/spaces and reserved device names (`CON`, `COM1`, …) are handled automatically.

## Install

Requirements:

- Go 1.23+
- Git available on your `PATH`

Using `go install`:

```bash
go install github.com/ciao22king/GoFetch@latest
```

Or build locally:

```bash
go build -o gofetch .
./gofetch
```

### Automatic installer (macOS / Linux)

```bash
chmod +x install.sh
./install.sh
```

The script builds only the files in this local directory, without using the network, into a temporary binary. It installs the stable name `gofetch` into `~/.local/bin`, adds that directory to your shell configuration (bash, zsh, fish or a POSIX fallback) and cleans up legacy timestamped binaries from older installers. The last local commit is only shown for information. The new binary is installed only after a successful build.

You can change the destination with:

```bash
GOFETCH_BIN_DIR="$HOME/.local/bin" ./install.sh
```

### Automatic installer (Windows)

```powershell
./install.ps1
```

It builds `gofetch.exe` into `%USERPROFILE%\.local\bin` (or `$env:GOFETCH_BIN_DIR`) and adds the folder to your user `PATH`. Open a new terminal afterwards.

## Usage

Interactive mode:

```bash
gofetch
```

Start with a URL:

```bash
gofetch --url git@github.com:owner/project.git
```

Clone a specific branch (skips the branch prompt):

```bash
gofetch --url https://github.com/owner/project --branch develop
```

Choose a parent directory:

```bash
gofetch --dir ~/Code
```

Clone without running the automatic build (useful for untrusted repositories):

```bash
gofetch --url https://github.com/owner/project --no-build
```

Shallow clone (only the latest commits — much faster on big repositories):

```bash
gofetch --url https://github.com/owner/project --depth 1
```

Non-interactive mode for scripts and CI (no TUI, destination path on stdout):

```bash
dest=$(gofetch --url https://github.com/owner/project --yes --no-build)
gofetch --url https://github.com/owner/project --yes --force   # replace an existing clone
```

Save a default clone directory (written to the config file):

```bash
gofetch --set-default-dir ~/Code
```

The config file lives next to the history (`~/.config/gofetch/config.json`, `%AppData%\gofetch\config.json` on Windows):

```json
{
  "default_dir": "~/Code",
  "depth": 0,
  "no_build": false
}
```

Command-line flags always override the config file.

## Keyboard shortcuts

| Key | Action |
| --- | --- |
| `Enter` | Continue / clone / open the selected fetch (empty branch = default) |
| `↑` / `↓` (or `k` / `j`) | Select a previous fetch |
| `n` | Start a new fetch |
| `u` | Update the selected clone (`git pull --ff-only`) |
| `/` or `Ctrl+L` | Search the history |
| `Tab` | Complete the branch from the remote / cycle recent destination folders |
| `s` | Toggle a shallow clone on the destination screen |
| `Ctrl+V` / `Cmd+V` | Paste a URL, branch or path from the clipboard |
| `Ctrl+O` | Open the selected folder in your file manager |
| `Ctrl+Y` | Copy the selected path to the clipboard |
| `Ctrl+D` | Remove the selected history entry |
| `r` | Reload the history from disk |
| `Esc` | Go back / cancel a running clone |
| `q` / `Ctrl+C` | Quit |

## Automatic builds

After a successful clone, GoFetch detects the project type and runs a build:

| Manifest | Command |
| --- | --- |
| `go.mod` | `go build ./...` |
| `Cargo.toml` | `cargo build` |
| `package.json` with a build script | `npm`/`yarn`/`pnpm run <script>` |
| `pom.xml` | `mvn -q -DskipTests package` |
| `build.gradle` / `build.gradle.kts` | `gradle build` |
| `Makefile` | `make` |
| `pyproject.toml` | `python3 -m build` |
| `setup.py` | `python3 setup.py build` |

Builds have a 10-minute timeout. If no supported setup is found, the clone still completes normally. Build scripts execute project-defined code, so use `--no-build` (or skip building) for repositories you do not trust.

## Development

```bash
go test ./...
go vet ./...
```

The test suite covers URL parsing, folder-name sanitization, history persistence, build detection, branch selection, remote-branch completion, pull summaries, config persistence, clone progress streaming, the TUI state machine and a real end-to-end `git clone`.

## Roadmap

- GitHub/GitLab API search
- Multiple repositories in one run
