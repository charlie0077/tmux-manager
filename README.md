# tmux-manager

A TUI tool for managing tmux sessions across multiple remote servers via SSH. Also supports local tmux.

## Features

- Browse and manage servers in a table view
- View, create, attach, and kill tmux sessions on any server
- Add/delete servers from the TUI (persisted to YAML config)
- Filter servers by name
- Local tmux support (auto-included)
- Ctrl-q to detach from attached sessions
- Remembers last selected server and session across restarts

## Install

### From source

Requires Go 1.22+:

```sh
go install github.com/charlie0077/tmux-manager/cmd/tmux-manager@latest
```

Then run:

```sh
tmux-manager
```

Or clone and build:

```sh
git clone https://github.com/charlie0077/tmux-manager.git
cd tmux-manager
go build -o tmux-manager ./cmd/tmux-manager
./tmux-manager
```

## Configuration

Config is stored at `~/.config/tmux-manager/config.yaml`:

```yaml
servers:
  - name: local
    host: local
  - name: prod-api
    host: user@prod.example.com
    key_file: ~/.ssh/id_rsa
```

On first run, an onboarding wizard helps you add your first server.

## Keybindings

### Server List

| Key   | Action              |
|-------|---------------------|
| enter | View sessions       |
| a     | Add server          |
| x     | Delete server       |
| r     | Refresh all servers |
| /     | Filter by name      |
| ?     | Toggle help         |
| q     | Quit                |

### Session Detail

| Key     | Action                 |
|---------|------------------------|
| enter   | Attach to session      |
| n       | New session            |
| x       | Kill session (confirm) |
| left/esc| Back to server list    |
| ?       | Toggle help            |

### While Attached

| Key    | Action                  |
|--------|-------------------------|
| Ctrl-q | Detach and return to TUI|

## License

MIT
