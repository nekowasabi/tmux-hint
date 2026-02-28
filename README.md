# tmux-hint

A lightweight Go plugin for tmux that overlays hints on visible pane text, letting you copy matches or jump to them in vim copy-mode. A minimal alternative to tmux-fingers.

## Features

- **Copy to clipboard** — auto-detects `pbcopy` (macOS), `wl-copy` (Wayland), or `xclip` (X11), and also writes to the tmux buffer
- **Vim-mode jump** — press `v` + a hint letter to enter tmux copy-mode and jump to the matched line
- **Overlay display** — rendered via `tmux display-popup`, no permanent pane splitting
- **Supported patterns** — URLs, file paths, IP addresses, UUIDs, git SHAs, numeric strings
- **Key bindings** — `a-z` to copy, `v`+hint to jump, `q` or `Esc` to cancel

## Requirements

- tmux 3.2+
- Go 1.21+

## Installation

### TPM

Add to your `~/.tmux.conf`:

```tmux
set -g @plugin 'takets/tmux-hint'
```

Then press `prefix + I` to install.

### run-shell (without TPM)

Clone the repo and build the binary:

```sh
git clone https://github.com/takets/tmux-hint ~/.tmux/plugins/tmux-hint
cd ~/.tmux/plugins/tmux-hint
go build -o bin/tmux-hint .
```

Add to your `~/.tmux.conf`:

```tmux
run-shell "~/.tmux/plugins/tmux-hint/tmux-hint.tmux"
```

Then reload the config:

```sh
tmux source-file ~/.tmux.conf
```

### Manual

Build the binary:

```sh
go build -o bin/tmux-hint .
```

Bind the binary directly in your `~/.tmux.conf`:

```tmux
bind-key F run-shell -b "/path/to/tmux-hint start #{pane_id}"
```

## Configuration

| Option | Default | Description |
|---|---|---|
| `@tmux-hint-key` | `F` | Key to trigger the hint overlay |

Example:

```tmux
set -g @tmux-hint-key 'F'
```

## Repository Layout

```
tmux-hint/
├── main.go         # entry point
├── capture.go      # tmux capture-pane wrapper
├── match.go        # regex pattern matching
├── hint.go         # hint label generation
├── display.go      # display-popup overlay
├── action.go       # copy / vim-jump actions
├── tmux-hint.tmux  # TPM entry point
└── go.mod
```

## Comparison with tmux-fingers

tmux-hint intentionally omits the following tmux-fingers features in exchange for simplicity and a single static binary:

- Multi-select mode
- Four modifier keys
- Custom action pipes
- Keyboard layout configuration
- Install wizard
- Crystal runtime dependency

## License

MIT
