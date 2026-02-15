# claude-usage

TUI for tracking your Claude rate limit usage. Built with [Charm](https://charm.sh).

![Go](https://img.shields.io/badge/Go-1.23-blue)

## Install

```bash
go install github.com/tnguyen21/claude-usage@latest
```

Or build from source:

```bash
git clone git@github.com:tnguyen21/claude-usage.git
cd claude-usage
go install .
```

## Usage

```bash
claude-usage
```

That's it. It reads your OAuth token from the macOS Keychain automatically (requires being logged into [Claude Code](https://docs.anthropic.com/en/docs/claude-code)).

### Compact mode

For tmux statusbars or scripts:

```bash
claude-usage --compact
# 5h:45% 7d:29%
```

```bash
# tmux example
set -g status-right '#(claude-usage --compact)'
```

### Environment variable

On Linux or if you want to use a specific token:

```bash
export CLAUDE_OAUTH_TOKEN="sk-ant-oat01-..."
claude-usage
```

## Keybindings

| Key | Action |
|-----|--------|
| `q` | Quit |
| `r` | Refresh |

Hover over a bar to see the exact utilization percentage.

## Requirements

- macOS (for Keychain auto-detection) or `CLAUDE_OAUTH_TOKEN` env var
- Logged into Claude Code (`claude`)
