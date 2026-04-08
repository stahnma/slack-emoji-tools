package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// WorkspaceConfig holds credentials and identity for a single Slack workspace.
type WorkspaceConfig struct {
	BotToken string `mapstructure:"bot_token"`
	Cookie   string `mapstructure:"cookie"`
	Team     string `mapstructure:"team"`
}

// Defaults holds default values for CLI behaviour.
type Defaults struct {
	OutputDir   string        `mapstructure:"output_dir"`
	UploadDelay time.Duration `mapstructure:"upload_delay"`
	AutoSuffix  bool          `mapstructure:"auto_suffix"`
}

// Config is the top-level configuration loaded from the config file and
// environment variables via Viper.
type Config struct {
	DefaultWorkspace string                      `mapstructure:"default_workspace"`
	Workspaces       map[string]WorkspaceConfig  `mapstructure:"workspaces"`
	Defaults         Defaults                    `mapstructure:"defaults"`
}

// Load reads the current Viper state into a typed Config struct.
func Load() (Config, error) {
	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return cfg, fmt.Errorf("unmarshalling config: %w", err)
	}
	return cfg, nil
}

// ResolveWorkspace returns the WorkspaceConfig for the given name.
// If name is empty it falls back to DefaultWorkspace, then to environment
// variables (SLACK_TOKEN, SLACK_COOKIE, SLACK_TEAM).
func (c Config) ResolveWorkspace(name string) (WorkspaceConfig, error) {
	// 1. Explicit name provided — look it up.
	if name != "" {
		ws, ok := c.Workspaces[name]
		if !ok {
			return WorkspaceConfig{}, fmt.Errorf("workspace %q not found in config", name)
		}
		return ws, nil
	}

	// 2. Fall back to default_workspace.
	if c.DefaultWorkspace != "" {
		ws, ok := c.Workspaces[c.DefaultWorkspace]
		if !ok {
			return WorkspaceConfig{}, fmt.Errorf("default workspace %q not found in config", c.DefaultWorkspace)
		}
		return ws, nil
	}

	// 3. Fall back to environment variables via Viper.
	token := viper.GetString("slack_token")
	cookie := viper.GetString("slack_cookie")
	team := viper.GetString("slack_team")

	if token == "" && cookie == "" {
		return WorkspaceConfig{}, fmt.Errorf("no workspace configured: set --workspace, default_workspace in config, or SLACK_TOKEN/SLACK_COOKIE env vars")
	}

	return WorkspaceConfig{
		BotToken: token,
		Cookie:   cookie,
		Team:     team,
	}, nil
}

// StripCookiePrefix removes the leading "d=" prefix from a Slack cookie
// value, if present.
func StripCookiePrefix(cookie string) string {
	return strings.TrimPrefix(cookie, "d=")
}
