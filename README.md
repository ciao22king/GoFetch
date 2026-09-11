# GoFetch

GoFetch is a small, focused terminal UI for cloning GitHub and GitLab repositories without remembering the exact `git clone` incantation.

![GoFetch terminal UI](https://placehold.co/1200x700/0f172a/e2e8f0?text=GoFetch)

## Why

The normal flow is simple, but it is still full of tiny decisions: HTTPS or SSH, where to put the folder, and whether the URL is even valid. GoFetch turns that into a calm flow and remembers every successful fetch:

1. Choose a previous fetch with `↑`/`↓` and press `Enter` to open its folder, or press `n` for a new clone.
2. Paste a GitHub or GitLab URL.
3. Choose the parent directory.
4. GoFetch runs `git clone` and saves the result in its local history.

It supports:

- GitHub and GitLab HTTPS URLs
- GitHub and GitLab SSH URLs
- GitLab nested groups
- `~` in destination paths
- Friendly validation and clone errors
- A keyboard-first Bubble Tea interface
- Persistent fetch history in `~/.config/gofetch/history.json`
- One-key opening of previously fetched repository folders

## Install

Requirements:

- Go 1.23+
- Git available on your `PATH`

```bash
gh repo clone ciao22king/GoFetch
```

Or build locally:

```bash
go build -o gofetch .
./gofetch
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
| `Esc` | Go back |
| `q` | Quit |

## Roadmap

- Clone destination preview
- Optional branch selection
- GitHub/GitLab API search
- Clone progress stream inside the TUI
