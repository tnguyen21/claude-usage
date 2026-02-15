# `claude-usage` — TUI for Claude Usage Tracking

A Go TUI built with [Charm](https://charm.sh) that shows your Claude rate limit usage as pretty terminal progress bars. Zero config — it reads your Claude Code credentials automatically.

---

## UX

```
$ go install github.com/tau/claude-usage@latest
$ claude-usage
```

That's it. The tool:

1. Reads your OAuth token from the macOS Keychain (`Claude Code-credentials`)
2. Hits the usage API
3. Shows animated progress bars

If you're not logged into Claude Code, it tells you:

```
 ✗ No Claude Code credentials found.
   Run "claude" and sign in first.
```

---

## API

### OAuth Token (from Claude Code Keychain)

```
# macOS Keychain entry:
security find-generic-password -s "Claude Code-credentials" -w
# Returns JSON:
# {
#   "claudeAiOauth": {
#     "accessToken": "sk-ant-oat01-...",
#     "refreshToken": "...",
#     "expiresAt": 1234567890,
#     "subscriptionType": "pro"
#   }
# }
```

```
GET https://api.anthropic.com/api/oauth/usage
Authorization: Bearer sk-ant-oat01-...
User-Agent: claude-code/2.0.32
anthropic-beta: oauth-2025-04-20
Accept: application/json
Content-Type: application/json
```

### Response Shape

```json
{
  "five_hour": {
    "utilization": 45.0,
    "resets_at": "2026-02-15T18:00:00+00:00"
  },
  "seven_day": {
    "utilization": 29.0,
    "resets_at": "2026-02-19T11:29:00+00:00"
  },
  "seven_day_opus": {
    "utilization": 0.0,
    "resets_at": null
  },
  "seven_day_oauth_apps": null,
  "iguana_necktie": null
}
```

```go
type UsageResponse struct {
    FiveHour         *UsageBucket    `json:"five_hour"`
    SevenDay         *UsageBucket    `json:"seven_day"`
    SevenDayOpus     *UsageBucket    `json:"seven_day_opus"`
    SevenDayOAuth    json.RawMessage `json:"seven_day_oauth_apps"`
    IguanaNecktie    json.RawMessage `json:"iguana_necktie"`
}

type UsageBucket struct {
    Utilization float64  `json:"utilization"`   // 0.0–100.0
    ResetsAt    *string  `json:"resets_at"`      // ISO 8601 or null
}
```

---

## Architecture

```
claude-usage/
├── go.mod
├── main.go              # entrypoint, credential loading, first fetch
├── keychain.go          # read OAuth token from macOS Keychain
├── client.go            # HTTP client for the usage API
├── model.go             # JSON types for API response
└── tui.go               # Bubble Tea model, view, update
```

Single package (`main`). Small enough that it doesn't need internal packages.

### Dependencies

```
github.com/charmbracelet/bubbletea       # TUI framework (Elm architecture)
github.com/charmbracelet/lipgloss        # Styling and layout
github.com/charmbracelet/bubbles/progress # Animated progress bars
github.com/charmbracelet/bubbles/spinner  # Loading spinner
github.com/charmbracelet/bubbles/key      # Keybinding helpers
```

No config libraries, no CLI flag libraries. Keep it minimal.

---

## Auth Flow

Single path — Claude Code OAuth from Keychain:

```
┌─────────────────────────────────┐
│ exec: security find-generic-    │
│ password -s "Claude Code-       │
│ credentials" -w                 │
└──────────────┬──────────────────┘
               │
               ▼
┌─────────────────────────────────┐
│ Parse JSON → extract            │
│ claudeAiOauth.accessToken       │
└──────────────┬──────────────────┘
               │
        ┌──────┴──────┐
        │             │
     found        not found
        │             │
        ▼             ▼
   fetch usage    print error,
   → show TUI     exit 1
```

Override with `CLAUDE_OAUTH_TOKEN` env var if needed (e.g. Linux, or manual token). That's the only config surface.

---

## TUI Layout

```
┌──────────────────────────────────────────────────────┐
│  claude-usage                          ↻ 4m30s   q ×│
│                                                      │
│  Session (5h)   ████████████░░░░░░░░░░░░░░  45%     │
│                 resets in 2h 47m                      │
│                                                      │
│  Weekly (7d)    ██████░░░░░░░░░░░░░░░░░░░░  29%     │
│                 resets Wed Feb 19                     │
│                                                      │
│  Opus (7d)     ░░░░░░░░░░░░░░░░░░░░░░░░░░   0%     │
│                 —                                     │
│                                                      │
│  Pro  •  updated 12:43                               │
└──────────────────────────────────────────────────────┘
```

The `bubbles/progress` component handles the bars. Lip Gloss handles the border, padding, and color theming.

### Bar Colors

| Utilization | Color        | Lip Gloss              |
|-------------|--------------|------------------------|
| 0–50%       | Green        | `lipgloss.Color("82")` |
| 50–75%      | Yellow       | `lipgloss.Color("220")`|
| 75–90%      | Orange       | `lipgloss.Color("208")`|
| 90–100%     | Red          | `lipgloss.Color("196")`|

Colors transition smoothly via `progress.WithGradient` or `progress.WithScaledGradient`.

### Keybindings

| Key | Action        |
|-----|---------------|
| `q` | Quit          |
| `r` | Force refresh |

That's it. Two keys. Keep it simple.

---

## Bubble Tea Model

```go
type model struct {
    // data
    usage    *UsageResponse
    err      error
    lastFetch time.Time
    stale    bool

    // UI components
    sessionBar progress.Model
    weeklyBar  progress.Model
    opusBar    progress.Model
    spinner    spinner.Model

    // state
    loading  bool
    width    int
    height   int
    token    string
}
```

### Messages

```go
type usageFetchedMsg struct {
    usage *UsageResponse
    err   error
}

type tickMsg time.Time
```

### Update Loop

```
Init
 └→ fetch usage (Cmd)
 └→ start tick (every 5 min)

Update
 ├─ usageFetchedMsg → store data, animate bars to new values
 ├─ tickMsg → re-fetch usage
 ├─ key "r" → re-fetch (with 10s debounce)
 ├─ key "q" → tea.Quit
 ├─ progress.FrameMsg → animate bar transitions
 └─ tea.WindowSizeMsg → resize bars
```

---

## Polling

- Default interval: **5 minutes**
- Use `tea.Tick` to schedule re-fetches
- Force refresh (`r`) debounced to 10 seconds minimum
- On network error: keep last data, mark `stale`, show a dim indicator

---

## Error Handling

| Scenario | Behavior |
|----------|----------|
| No keychain entry | Print friendly message, exit 1 |
| Token expired / 401 | Show in TUI: "Token expired — re-login to Claude Code" |
| Network error | Keep last data, show "stale" badge, retry on next tick |
| Unknown JSON fields | Ignored (Go's `json.Unmarshal` does this by default) |

---

## Compact Mode

For tmux / statusbar integration:

```
$ claude-usage --compact
5h:45% 7d:29%
```

One fetch, one line, exit. No TUI. Use with:

```bash
# tmux
set -g status-right '#(claude-usage --compact)'
```

This is the only CLI flag. Use `os.Args` directly — no need for a flag library.

---

## Stretch Goals

- **Linux keychain support:** `secret-tool` / `libsecret` for GNOME Keyring
- **Notifications:** desktop notification at 75%, 90% thresholds
- **History sparkline:** `bubbles/sparkline` showing usage over time
- **Adaptive colors:** `lipgloss.AdaptiveColor` for light/dark terminal themes

---

## Prior Art

| Project | Notes |
|---------|-------|
| [ClaudeUsageBar](https://github.com/Artzainnn/ClaudeUsageBar) | macOS menu bar, Swift, cookie auth |
| [Claude-Usage-Tracker](https://github.com/hamed-elfayome/Claude-Usage-Tracker) | macOS, multi-profile |
| [CodexBar](https://github.com/steipete/CodexBar) | Multi-provider, Linux CLI |
| [codelynx statusline](https://codelynx.dev/posts/claude-code-usage-limits-statusline) | OAuth path discovery, TypeScript |

---

## Caveats

1. **Unofficial endpoint.** `api.anthropic.com/api/oauth/usage` is an undocumented internal API. It can break at any time.
2. **macOS-only by default.** Keychain auto-detection only works on macOS. Linux users need `CLAUDE_OAUTH_TOKEN` env var.
3. **Rate limit ambiguity.** The `utilization` field is a percentage but Anthropic doesn't disclose what the actual token budgets are.
