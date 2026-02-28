#!/usr/bin/env bash
PLUGIN_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BINARY="$PLUGIN_DIR/bin/tmux-hint"

if [ ! -f "$BINARY" ]; then
  tmux display-message "tmux-hint: run 'cd $PLUGIN_DIR && go build -o bin/tmux-hint .'"
  exit 1
fi

# Default keybinding: prefix + F
BIND_KEY=$(tmux show-option -gv @tmux-hint-key 2>/dev/null || echo "F")
tmux bind-key "$BIND_KEY" run-shell -b "$BINARY start #{pane_id}"
"$BINARY" load-config
