# GoFetch

GoFetch is a small, focused terminal UI for cloning GitHub and GitLab repositories without remembering the exact `git clone` incantation.

![GoFetch terminal UI](https://placehold.co/1200x700/0f172a/e2e8f0?text=GoFetch)

## Why

The normal flow is simple, but it is still full of tiny decisions: HTTPS or SSH, where to put the folder, and whether the URL is even valid. GoFetch turns that into a calm two-step flow:

1. Paste a GitHub or GitLab URL.
2. Choose the parent directory.
3. GoFetch runs `git clone` and reports the result.

It supports:

- GitHub and GitLab HTTPS URLs
- GitHub and GitLab SSH URLs
- GitLab nested groups
- `~` in destination paths
- Friendly validation and clone errors
- A keyboard-first Bubble Tea interface

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
| `Esc` | Go back |
| `q` | Quit |

## Roadmap

- Recent repositories
- Clone destination preview
- Optional branch selection
- GitHub/GitLab API search
- Clone progress stream inside the TUI