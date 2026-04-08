# slack-emoji-tools

[![CI](https://github.com/stahnma/slack-emoji-tools/actions/workflows/ci.yml/badge.svg)](https://github.com/stahnma/slack-emoji-tools/actions/workflows/ci.yml)

A CLI for managing custom Slack emoji. Download, list, and bulk-upload emoji across workspaces -- including free-tier Slack plans.

## Install

```bash
go install github.com/stahnma/slack-emoji-tools@latest
```

Or build from source:

```bash
make build
```

## Quick Start

Run the setup wizard to configure your first workspace:

```bash
slack-emoji-tools init
```

Then try it out:

```bash
# List all custom emoji
slack-emoji-tools list

# Download all emoji to your local machine
slack-emoji-tools grab

# Upload emoji from a directory
slack-emoji-tools upload ./my-emoji/
```

## Getting Credentials

`slack-emoji-tools` uses two types of credentials depending on the operation:

### Bot token (for `grab` and `list`)

A Slack Bot User OAuth Token with the `emoji:read` scope. This is stable and long-lived.

1. Go to your [Slack App settings](https://api.slack.com/apps) and create an app (or use an existing one).
2. Under **OAuth & Permissions**, add the `emoji:read` scope.
3. Install the app to your workspace.
4. Copy the **Bot User OAuth Token** (starts with `xoxb-`).

### Browser cookie (for `upload` and `resolve`)

Upload uses Slack's browser-based emoji endpoint, which requires a session cookie. This works on free-tier workspaces without admin API access.

1. Open your Slack workspace in a browser (e.g., `https://mycompany.slack.com`).
2. Open DevTools (F12 or Cmd+Option+I on Mac).
3. Go to **Application** (Chrome) or **Storage** (Firefox) tab.
4. Under **Cookies**, find the cookie named `d` and copy its value (starts with `xoxd-`).

The API token (`xoxc-*`) is derived automatically from the cookie -- you don't need to extract it yourself.

> **Note:** Session cookies expire every 1-2 weeks. When you get authentication errors, refresh with:
> ```bash
> slack-emoji-tools init --update-cookie
> ```

## Configuration

The `init` wizard writes a YAML config file to `~/.config/slack-emoji-tools/config.yaml`. You can also create it manually:

```yaml
default_workspace: mycompany

workspaces:
  mycompany:
    bot_token: xoxb-...
    cookie: xoxd-...
    team: mycompany

defaults:
  output_dir: ./emojis
  upload_delay: 1s
```

Multiple workspaces are supported. Switch between them with `--workspace`:

```bash
slack-emoji-tools list --workspace othercompany
```

Credentials can also be set via environment variables for quick, single-workspace usage:

```bash
export SLACK_TOKEN="xoxb-..."
export SLACK_COOKIE="xoxd-..."
export SLACK_TEAM="mycompany"
```

Precedence: flags > environment variables > config file > defaults.

## Commands

### `grab` -- Download emoji

Downloads all custom emoji from a workspace.

```bash
# Download to default location (~/.config/slack-emoji-tools/emojis/{team}/)
slack-emoji-tools grab

# Download to a specific directory
slack-emoji-tools grab --output ./my-emojis

# Include alias emoji (skipped by default)
slack-emoji-tools grab --skip-aliases=false
```

### `list` -- List emoji

Lists custom emoji names. No files are written.

```bash
# Plain text (one per line)
slack-emoji-tools list

# Include aliases
slack-emoji-tools list --include-aliases

# JSON output
slack-emoji-tools list --format json

# CSV output
slack-emoji-tools list --format csv
```

### `upload` -- Bulk upload

Uploads emoji from a directory of image files (.png, .gif, .jpg, .jpeg). Tracks progress so interrupted runs can be safely resumed.

```bash
# Upload all emoji from a directory
slack-emoji-tools upload ./emoji/

# Preview what would be uploaded
slack-emoji-tools upload --dry-run ./emoji/

# Auto-suffix on name conflicts (e.g., wave2, wave3)
slack-emoji-tools upload --auto-suffix ./emoji/

# Slower uploads (be kind to the API)
slack-emoji-tools upload --delay 2s ./emoji/
```

When a name conflict occurs, `slack-emoji-tools` checks if the existing emoji is identical (SHA256 comparison). If the images match, it's treated as already uploaded. Genuine conflicts are recorded for later resolution.

### `status` -- Upload progress

Shows how many emoji have been uploaded, how many have conflicts, and how many remain.

```bash
slack-emoji-tools status ./emoji/
```

### `resolve` -- Fix conflicts

Interactively walk through name conflicts and choose new names.

```bash
slack-emoji-tools resolve ./emoji/
```

### `init` -- Setup wizard

Configure workspaces and credentials interactively.

```bash
# Full setup wizard
slack-emoji-tools init

# Quick cookie refresh
slack-emoji-tools init --update-cookie
```

## Migrating Emoji Between Workspaces

Grab from one workspace, upload to another:

```bash
slack-emoji-tools grab --workspace source --output ./transfer/
slack-emoji-tools upload --workspace destination ./transfer/
```

## Global Flags

| Flag | Description |
|------|-------------|
| `--config` | Explicit config file path |
| `--workspace` | Override default workspace |
| `--verbose` | Enable debug logging |

## Notes

- Session cookies expire periodically. If you get auth errors, run `slack-emoji-tools init --update-cookie`.
- Upload uses Slack's undocumented `emoji.add` endpoint (the same one the web UI uses). It works on free-tier workspaces.
- Rate limits are handled automatically with exponential backoff (2s, 4s, 8s, ..., up to 60s).
- Upload state is tracked per-workspace in `~/.config/slack-emoji-tools/state/{team}/`. Re-running upload is safe and idempotent.
- Be respectful of rate limits. The default 1-second delay between uploads is a reasonable starting point.

## Development

```bash
make help       # Show available targets
make build      # Build the binary
make test       # Run tests
make lint       # Run vet + gofmt
```

See [DEVELOPMENT.md](DEVELOPMENT.md) for project structure and architecture details.

## Like this project?

If you find `slack-emoji-tools` useful, give it a star on GitHub -- it helps others discover the project!

## License

MIT
