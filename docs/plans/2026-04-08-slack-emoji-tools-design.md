# slack-emoji-tools — Unified Design

**Date:** 2026-04-08
**Status:** Draft

## Overview

Combine `slack-emoji-grabber` and `slack-emoji-uploader` into a single CLI tool called `slack-emoji-tools`. One binary, one config file, unified UX across all emoji operations.

## Commands

### Initial Build

| Command | Purpose | Auth Required |
|---------|---------|---------------|
| `init` | Interactive setup wizard | none |
| `grab` | Download all custom emoji from a workspace | bot token (`xoxb-*`) |
| `list` | List custom emoji names | bot token (`xoxb-*`) |
| `upload <dir>` | Bulk upload emoji from a directory | cookie + team |
| `status` | Show upload progress for a workspace | none (reads local state) |
| `resolve` | Interactive conflict resolution | cookie + team |

### Future Commands (designed for, not implemented)

| Command | Purpose |
|---------|---------|
| `delete` | Remove custom emoji by name |
| `alias` | Create an alias for an existing emoji |
| `sync` | Grab from workspace A, upload to workspace B |
| `diff` | Compare emoji between two workspaces or local directory vs workspace |

## Project Structure

```
slack-emoji-tools/
├── go.mod                          # github.com/stahnma/slack-emoji-tools
├── go.sum
├── Makefile
├── README.md
├── main.go                         # cobra entry point
├── cmd/
│   ├── root.go                     # root command, Viper config loading
│   ├── init.go                     # init subcommand (setup wizard)
│   ├── grab.go                     # grab subcommand
│   ├── list.go                     # list subcommand
│   ├── upload.go                   # upload subcommand
│   ├── status.go                   # status subcommand
│   └── resolve.go                  # resolve subcommand
├── internal/
│   ├── config/
│   │   └── config.go               # Viper setup, config file search, typed config struct
│   ├── grabber/
│   │   ├── grabber.go              # download logic (from existing grabber)
│   │   └── grabber_test.go
│   ├── uploader/
│   │   ├── client.go               # custom HTTP client for emoji.add
│   │   └── client_test.go
│   ├── state/
│   │   ├── state.go                # state + conflict JSON persistence
│   │   └── state_test.go
│   └── walker/
│       ├── walker.go               # directory walking, name derivation
│       └── walker_test.go
└── docs/
    └── plans/
```

## Configuration

### Stack

Cobra + Viper for CLI and config management. No `.env` / godotenv.

### Config File Search Order

1. `--config` flag (explicit override)
2. `./slack-emoji-tools.yaml` (current directory)
3. `~/.config/slack-emoji-tools/config.yaml` (XDG)
4. `~/.slack-emoji-tools.yaml` (home dotfile)

### Precedence

flags > env vars > config file > defaults

### Config File Format

```yaml
default_workspace: mycompany

workspaces:
  mycompany:
    bot_token: xoxb-...
    cookie: xoxd-...
    team: mycompany
  othercompany:
    bot_token: xoxb-...
    cookie: xoxd-...
    team: othercompany

defaults:
  output_dir: ./emojis
  upload_delay: 1s
  auto_suffix: false
```

### Environment Variables

Apply to the active workspace (for single-workspace / quick usage):

- `SLACK_TOKEN` — bot token
- `SLACK_COOKIE` — browser cookie
- `SLACK_TEAM` — team subdomain

### Global Flags

- `--config` — explicit config file path
- `--workspace` — override `default_workspace`
- `--verbose` — enable slog debug output

## Authentication

### Auth per Command

- **Read operations** (`grab`, `list`): Bot token (`xoxb-*`) with `emoji:read` scope. Long-lived, stable.
- **Write operations** (`upload`, `resolve`): Browser cookie (`xoxd-*`) + team subdomain. Cookie is used to auto-derive `xoxc-*` session token. Expires every 1-2 weeks.

### Setup Wizard (`init`)

Interactive walkthrough that builds the config file:

```
$ slack-emoji-tools init

Welcome to slack-emoji-tools setup!

Workspace name (e.g. mycompany): mycompany

── Grab/List credentials (optional) ──
These use a Slack Bot Token with emoji:read scope.
Bot token (xoxb-...): xoxb-1234-abcd

── Upload credentials (optional) ──
These use a browser session cookie from Slack.
See: https://github.com/stahnma/slack-emoji-tools#getting-credentials
Browser cookie (xoxd-...): xoxd-5678-efgh
Team subdomain [mycompany]: mycompany

── Defaults ──
Output directory for grab [./emojis]:
Upload delay [1s]:

Config written to ~/.config/slack-emoji-tools/config.yaml

Add another workspace? (y/n): n

You're all set! Try: slack-emoji-tools list
```

Behaviors:
- Fields can be left blank (Enter to skip)
- Team subdomain defaults to the workspace name
- Re-running `init` detects existing config, offers to add a workspace or edit existing
- Validates token format (prefix check), warns if wrong
- Credentials masked in terminal output

### Cookie Expiration Handling

- `init --update-cookie` — fast path to replace just the cookie without full wizard
- Auth errors (`not_authed`, `invalid_auth`, `token_revoked`) print a clear message directing user to `init --update-cookie`
- Auto-prompt on first run if no config file found

## State Management

### Location

```
~/.config/slack-emoji-tools/state/{team}/
├── emoji-state.json
└── emoji-conflicts.json
```

Keyed by team name. Supports multiple workspaces.

### State File (`emoji-state.json`)

Tracks successfully uploaded emoji for idempotency:

```json
{
  "emoji/partyparrot.gif": {
    "name": "partyparrot",
    "uploaded_at": "2026-04-08T03:57:29Z"
  }
}
```

### Conflicts File (`emoji-conflicts.json`)

Records emoji that failed due to genuine name collisions (after SHA256 check):

```json
{
  "tada-ish.png": {
    "name": "tada-ish",
    "error": "error_name_taken",
    "attempted": ["tada-ish"],
    "last_attempt": "2026-04-08T03:57:29Z"
  }
}
```

## Downloaded Emoji Location

Default: `~/.config/slack-emoji-tools/emojis/{team}/`

Override with `--output` flag or `defaults.output_dir` in config file.

## Smart Duplicate Detection

When upload gets a name conflict (`error_name_taken`):

1. Call `emoji.list` to get the URL of the existing remote emoji
2. Download it to a temp file
3. Compare SHA256 of local file vs remote file
4. **Match** — same image, treat as already uploaded. Record in state. No conflict.
5. **Mismatch** — genuine conflict. Record in `emoji-conflicts.json`.

This reduces noise in the conflicts file, especially for sync-like workflows.

## Command Details

### `grab`

- Downloads all custom emoji from the workspace
- Auth: bot token
- Flags: `--output` (default from config or `~/.config/slack-emoji-tools/emojis/{team}/`), `--skip-aliases` (default true)
- Skips already-downloaded files
- Uses official Slack SDK (`slack-go/slack`) for `emoji.list`

### `list`

- Lists all custom emoji names in the workspace
- Auth: bot token
- Flags: `--format` (`text`, `json`, `csv`), `--include-aliases` (default false)
- No file I/O — prints to stdout

### `upload <directory>`

- Bulk uploads emoji from a directory
- Auth: cookie + team
- Flags: `--auto-suffix`, `--delay` (default 1s), `--dry-run`
- Uses undocumented `emoji.add` endpoint (works on free Slack plans)
- Exponential backoff on rate limits (2s, 4s, 8s, ..., max 60s)
- SHA256 duplicate check on name conflicts
- State tracked in `~/.config/slack-emoji-tools/state/{team}/`

### `status`

- Shows upload progress: total, uploaded, conflicts, remaining
- Auth: none (reads local state files)
- Flags: `--workspace` to check a different workspace

### `resolve`

- Walks through `emoji-conflicts.json` interactively
- Prompts for new name for each conflict (or skip)
- Auth: cookie + team

## Logging

- `slog` for internal packages (`internal/`) — structured, leveled
- `fmt` for user-facing output in `cmd/` — clean CLI output
- `--verbose` enables slog debug output

## Dependencies

- `github.com/spf13/cobra` — CLI framework
- `github.com/spf13/viper` — config file + env var management
- `github.com/slack-go/slack` — official SDK for grab/list

## Build

- Single `go.mod` at root
- `CGO_ENABLED=0` for static binaries
- Version embedded via `ldflags` from `git describe`
- `go install github.com/stahnma/slack-emoji-tools@latest`

Standard Makefile targets: `build`, `install`, `test`, `lint`, `tidy`, `clean`.

## Migration

No migration path from existing tools. Clean break. State files from `slack-emoji-uploader` can be manually copied to `~/.config/slack-emoji-tools/state/{team}/` if desired, but upload is idempotent so re-running is safe.

## Slack API Operations

| Operation | Endpoint | Method | Used By |
|-----------|----------|--------|---------|
| List emoji | `emoji.list` (official SDK) | SDK call | `grab`, `list`, duplicate detection |
| Download emoji | HTTP GET on emoji URL | GET | `grab`, duplicate detection |
| Upload emoji | `{team}.slack.com/api/emoji.add` (undocumented) | POST multipart | `upload`, `resolve` |
| Derive token | Fetch `{team}.slack.com`, extract `xoxc-*` | GET + regex | `upload`, `resolve` (auto) |
