# How to Get a Slack Bot Token

This guide walks you through creating a Slack App and obtaining a Bot Token
(`xoxb-...`) with the `emoji:read` scope, which is needed by `slack-emoji-tools`
to list and download emoji from your workspace.

## Steps

### 1. Create a Slack App

1. Go to <https://api.slack.com/apps> and sign in.
2. Click **Create New App**.
3. Choose **From scratch**.
4. Give your app a name (e.g. "Emoji Tools") and select the workspace you want
   to use it with.
5. Click **Create App**.

### 2. Add Bot Token Scopes

1. In the left sidebar, click **OAuth & Permissions**.
2. Scroll down to **Scopes > Bot Token Scopes**.
3. Click **Add an OAuth Scope** and add:
   - `emoji:read` — allows the bot to list custom emoji in the workspace.
4. If you plan to use other features in the future, you can add more scopes
   later.

### 3. Install the App to Your Workspace

1. Scroll back up on the **OAuth & Permissions** page.
2. Click **Install to Workspace** (or **Reinstall to Workspace** if updating
   scopes).
3. Review the permissions and click **Allow**.

### 4. Copy the Bot Token

After installation, you will see a **Bot User OAuth Token** on the
**OAuth & Permissions** page. It starts with `xoxb-`.

Copy this token and paste it into the `slack-emoji-tools init` wizard when
prompted.

## Security Notes

- Treat the bot token like a password. Do not commit it to version control.
- `slack-emoji-tools` stores the token in `~/.config/slack-emoji-tools/config.yaml`
  with file permissions restricted to your user.
- You can revoke the token at any time by uninstalling the app from the
  [Slack App Management](https://api.slack.com/apps) page.
