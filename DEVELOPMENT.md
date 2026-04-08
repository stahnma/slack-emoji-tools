# Development

## Project Structure

```
slack-emoji-tools/
├── main.go                         # Cobra entry point
├── cmd/
│   ├── root.go                     # Root command, Viper config loading
│   ├── init.go                     # Interactive setup wizard
│   ├── grab.go                     # Download emoji from workspace
│   ├── list.go                     # List emoji names
│   ├── upload.go                   # Bulk upload with state tracking
│   ├── status.go                   # Show upload progress
│   └── resolve.go                  # Interactive conflict resolution
├── internal/
│   ├── config/
│   │   └── config.go               # Viper integration, typed config struct
│   ├── grabber/
│   │   ├── grabber.go              # Download logic, SlackClient interface
│   │   └── grabber_test.go
│   ├── uploader/
│   │   ├── client.go               # HTTP client for emoji.add endpoint
│   │   ├── client_test.go
│   │   ├── dedup.go                # SHA256 duplicate detection
│   │   └── dedup_test.go
│   ├── state/
│   │   ├── state.go                # Upload state + conflict persistence
│   │   └── state_test.go
│   └── walker/
│       ├── walker.go               # Directory walking, name derivation
│       └── walker_test.go
├── docs/
│   └── plans/                      # Design documents
├── Makefile
├── go.mod
└── go.sum
```

## Architecture

- **cmd/** -- Cobra commands. User-facing output uses `fmt`. Each command loads config via `internal/config`, resolves the workspace, then delegates to the appropriate `internal/` package.
- **internal/config** -- Typed config struct with Viper integration. Handles workspace resolution (explicit name > default_workspace > env vars).
- **internal/grabber** -- Downloads emoji using the official `slack-go/slack` SDK (`emoji.list` API). Accepts a `SlackClient` interface for testability.
- **internal/uploader** -- Uploads emoji via Slack's undocumented `emoji.add` multipart endpoint. Handles rate limiting with exponential backoff. Also contains SHA256 dedup logic and `FetchToken` for deriving `xoxc-*` tokens from session cookies.
- **internal/state** -- JSON persistence for upload state (`emoji-state.json`) and conflicts (`emoji-conflicts.json`). Team-scoped directories under `~/.config/slack-emoji-tools/state/{team}/`. Atomic writes via temp file + rename.
- **internal/walker** -- Recursively finds image files (.png, .gif, .jpg, .jpeg) and derives emoji names from filenames.

## Logging

- `internal/` packages use `log/slog` for structured, leveled logging.
- `cmd/` uses `fmt` for clean user-facing output.
- `--verbose` enables slog debug output on stderr.

## Slack API Operations

| Operation | Endpoint | Used By |
|-----------|----------|---------|
| List emoji | `emoji.list` (official SDK) | grab, list, dedup |
| Download emoji | HTTP GET on emoji URL | grab, dedup |
| Upload emoji | `{team}.slack.com/api/emoji.add` (undocumented) | upload, resolve |
| Derive token | Fetch `{team}.slack.com`, extract `xoxc-*` | upload, resolve |

## Authentication

- **Read operations** (grab, list): Bot token (`xoxb-*`) with `emoji:read` scope.
- **Write operations** (upload, resolve): Browser cookie (`xoxd-*`) + team subdomain. The `xoxc-*` session token is derived automatically.

## Config File Search Order

1. `--config` flag (explicit path)
2. `./config.yaml` (current directory)
3. `~/.config/slack-emoji-tools/config.yaml` (XDG)
4. `~/config.yaml` (home directory)

## Running Tests

```bash
make test           # Full test suite
make test-short     # Short mode
make vet            # go vet
make fmt            # gofmt
```
