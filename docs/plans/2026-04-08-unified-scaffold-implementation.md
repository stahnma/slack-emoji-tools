# Unified slack-emoji-tools Scaffold — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Scaffold the unified `slack-emoji-tools` CLI with Cobra + Viper, porting all existing code from the two legacy tools into the new project structure described in the design doc.

**Architecture:** Single Go module at the repo root with `cmd/` for Cobra commands and `internal/` for business logic. Config via Viper (YAML files + env vars). Existing grabber, uploader/client, state, and walker code is moved into `internal/` packages with minimal changes. The `init` command is deferred to a later task.

**Tech Stack:** Go 1.26+, Cobra, Viper, slack-go/slack, slog

---

### Task 1: Initialize Go module and dependencies

**Files:**
- Create: `go.mod`
- Create: `main.go`

**Step 1: Create go.mod**

```bash
cd /Users/stahnma/go/src/github.com/stahnma/slack-emoji-tools
go mod init github.com/stahnma/slack-emoji-tools
```

**Step 2: Add dependencies**

```bash
go get github.com/spf13/cobra@latest
go get github.com/spf13/viper@latest
go get github.com/slack-go/slack@latest
```

**Step 3: Create main.go**

```go
package main

import (
	"fmt"
	"os"

	"github.com/stahnma/slack-emoji-tools/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
```

**Step 4: Verify it compiles (will fail until cmd/ exists — that's expected)**

Move to Task 2 immediately.

**Step 5: Commit**

```bash
git add go.mod go.sum main.go
git commit -m "feat: initialize go module with cobra/viper/slack-go dependencies"
```

---

### Task 2: Create root command with Viper config loading

**Files:**
- Create: `cmd/root.go`

**Step 1: Create cmd/root.go**

```go
package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile   string
	workspace string
	verbose   bool
)

var version = "dev"

var rootCmd = &cobra.Command{
	Use:   "slack-emoji-tools",
	Short: "Manage custom Slack emoji — grab, list, upload, and more",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if verbose {
			slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug})))
		}
	},
	SilenceUsage:  true,
	SilenceErrors: true,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file path")
	rootCmd.PersistentFlags().StringVar(&workspace, "workspace", "", "workspace name (overrides default_workspace)")
	rootCmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "enable debug logging")
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		viper.SetConfigName("slack-emoji-tools")
		viper.SetConfigType("yaml")
		viper.AddConfigPath(".")
		if xdg, err := os.UserConfigDir(); err == nil {
			viper.AddConfigPath(filepath.Join(xdg, "slack-emoji-tools"))
		}
		if home, err := os.UserHomeDir(); err == nil {
			viper.AddConfigPath(home)
			viper.SetConfigName(".slack-emoji-tools")
		}
	}

	viper.SetEnvPrefix("")
	viper.BindEnv("slack_token", "SLACK_TOKEN")
	viper.BindEnv("slack_cookie", "SLACK_COOKIE")
	viper.BindEnv("slack_team", "SLACK_TEAM")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			fmt.Fprintf(os.Stderr, "Warning: error reading config: %v\n", err)
		}
	}
}
```

**Step 2: Verify it compiles**

```bash
go build ./...
```

**Step 3: Commit**

```bash
git add cmd/root.go main.go go.mod go.sum
git commit -m "feat: add root command with Viper config loading"
```

---

### Task 3: Create internal/config package

**Files:**
- Create: `internal/config/config.go`

**Step 1: Create the typed config struct and workspace resolver**

```go
package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// WorkspaceConfig holds credentials for a single workspace.
type WorkspaceConfig struct {
	BotToken string `mapstructure:"bot_token"`
	Cookie   string `mapstructure:"cookie"`
	Team     string `mapstructure:"team"`
}

// Defaults holds default values for commands.
type Defaults struct {
	OutputDir   string `mapstructure:"output_dir"`
	UploadDelay string `mapstructure:"upload_delay"`
	AutoSuffix  bool   `mapstructure:"auto_suffix"`
}

// Config is the top-level configuration.
type Config struct {
	DefaultWorkspace string                     `mapstructure:"default_workspace"`
	Workspaces       map[string]WorkspaceConfig `mapstructure:"workspaces"`
	Defaults         Defaults                   `mapstructure:"defaults"`
}

// Load reads the Viper state into a typed Config.
func Load() (*Config, error) {
	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}
	return &cfg, nil
}

// ResolveWorkspace returns the workspace config for the given name,
// falling back to default_workspace, then to env vars.
func (c *Config) ResolveWorkspace(name string) (*WorkspaceConfig, error) {
	if name == "" {
		name = c.DefaultWorkspace
	}

	if name != "" && c.Workspaces != nil {
		if ws, ok := c.Workspaces[name]; ok {
			return &ws, nil
		}
		return nil, fmt.Errorf("workspace %q not found in config", name)
	}

	// Fall back to env vars
	ws := &WorkspaceConfig{
		BotToken: viper.GetString("slack_token"),
		Cookie:   viper.GetString("slack_cookie"),
		Team:     viper.GetString("slack_team"),
	}
	if ws.BotToken == "" && ws.Cookie == "" {
		return nil, fmt.Errorf("no workspace configured — run 'slack-emoji-tools init' or set SLACK_TOKEN")
	}
	return ws, nil
}

// StripCookiePrefix removes the "d=" prefix if present.
func StripCookiePrefix(cookie string) string {
	return strings.TrimPrefix(cookie, "d=")
}
```

**Step 2: Verify it compiles**

```bash
go build ./...
```

**Step 3: Commit**

```bash
git add internal/config/config.go
git commit -m "feat: add typed config struct with workspace resolution"
```

---

### Task 4: Port internal/grabber package

**Files:**
- Create: `internal/grabber/grabber.go`
- Create: `internal/grabber/grabber_test.go`

**Step 1: Copy and adapt grabber.go from legacy**

Port `slack-emoji-grabber/grabber.go` into `internal/grabber/grabber.go`. The code is nearly identical — just change the package name to `grabber` and keep the `SlackClient` interface as-is.

```go
// internal/grabber/grabber.go
// Copy from slack-emoji-grabber/grabber.go verbatim — it's already well-structured.
// Only change: package declaration is already `grabber`, so just copy it.
```

**Step 2: Copy and adapt grabber_test.go from legacy**

Port `slack-emoji-grabber/grabber_test.go` into `internal/grabber/grabber_test.go`. Same package, same tests.

**Step 3: Run tests**

```bash
go test ./internal/grabber/ -v
```

Expected: all tests pass.

**Step 4: Commit**

```bash
git add internal/grabber/
git commit -m "feat: port grabber package from slack-emoji-grabber"
```

---

### Task 5: Port internal/uploader package (was internal/slack)

**Files:**
- Create: `internal/uploader/client.go`
- Create: `internal/uploader/client_test.go`

**Step 1: Copy and adapt client.go**

Port `slack-emoji-uploader/internal/slack/client.go` to `internal/uploader/client.go`. Changes:
- Package name: `uploader`
- Replace `fmt.Printf` calls with `slog.Debug` for verbose output
- Replace `fmt.Printf("rate limited...")` with `slog.Info`

**Step 2: Copy and adapt client_test.go**

Port `slack-emoji-uploader/internal/slack/client_test.go` to `internal/uploader/client_test.go`. Change package name to `uploader`.

**Step 3: Run tests**

```bash
go test ./internal/uploader/ -v
```

Expected: all tests pass.

**Step 4: Commit**

```bash
git add internal/uploader/
git commit -m "feat: port uploader client from slack-emoji-uploader"
```

---

### Task 6: Port internal/state package

**Files:**
- Create: `internal/state/state.go`
- Create: `internal/state/state_test.go`

**Step 1: Copy state.go**

Port `slack-emoji-uploader/internal/state/state.go` to `internal/state/state.go`. Changes:
- Update `Load` and `LoadConflicts` to accept a base directory, constructing full paths internally. This supports the design doc's `~/.config/slack-emoji-tools/state/{team}/` pattern.

```go
// Key change: Load takes a directory, not a file path.
// The state files are always named emoji-state.json and emoji-conflicts.json.

func Load(dir string) (*State, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("creating state directory: %w", err)
	}
	path := filepath.Join(dir, "emoji-state.json")
	// ... rest same as before
}

func LoadConflicts(dir string) (*Conflicts, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("creating state directory: %w", err)
	}
	path := filepath.Join(dir, "emoji-conflicts.json")
	// ... rest same as before
}

// StateDir returns the state directory for a team.
func StateDir(team string) (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("finding config dir: %w", err)
	}
	return filepath.Join(configDir, "slack-emoji-tools", "state", team), nil
}
```

**Step 2: Copy and adapt state_test.go**

Update tests to use directory-based loading.

**Step 3: Run tests**

```bash
go test ./internal/state/ -v
```

Expected: all tests pass.

**Step 4: Commit**

```bash
git add internal/state/
git commit -m "feat: port state package with team-scoped directories"
```

---

### Task 7: Port internal/walker package

**Files:**
- Create: `internal/walker/walker.go`
- Create: `internal/walker/walker_test.go`

**Step 1: Copy walker.go verbatim**

Port `slack-emoji-uploader/internal/walker/walker.go` to `internal/walker/walker.go`. No changes needed.

**Step 2: Copy walker_test.go verbatim**

Port `slack-emoji-uploader/internal/walker/walker_test.go` to `internal/walker/walker_test.go`. No changes needed.

**Step 3: Run tests**

```bash
go test ./internal/walker/ -v
```

Expected: all tests pass.

**Step 4: Commit**

```bash
git add internal/walker/
git commit -m "feat: port walker package from slack-emoji-uploader"
```

---

### Task 8: Create `grab` command

**Files:**
- Create: `cmd/grab.go`

**Step 1: Create cmd/grab.go**

```go
package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"

	"github.com/slack-go/slack"
	"github.com/spf13/cobra"
	"github.com/stahnma/slack-emoji-tools/internal/config"
	"github.com/stahnma/slack-emoji-tools/internal/grabber"
)

var grabOutput string
var grabSkipAliases bool

var grabCmd = &cobra.Command{
	Use:   "grab",
	Short: "Download all custom emoji from a workspace",
	RunE:  runGrab,
}

func init() {
	grabCmd.Flags().StringVar(&grabOutput, "output", "", "output directory (default: ~/.config/slack-emoji-tools/emojis/{team}/)")
	grabCmd.Flags().BoolVar(&grabSkipAliases, "skip-aliases", true, "skip alias emoji")
	rootCmd.AddCommand(grabCmd)
}

func runGrab(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	ws, err := cfg.ResolveWorkspace(workspace)
	if err != nil {
		return err
	}
	if ws.BotToken == "" {
		return fmt.Errorf("bot token required for grab — set bot_token in config or SLACK_TOKEN env var")
	}

	outputDir := grabOutput
	if outputDir == "" && cfg.Defaults.OutputDir != "" {
		outputDir = cfg.Defaults.OutputDir
	}
	if outputDir == "" {
		configDir, err := os.UserConfigDir()
		if err != nil {
			return fmt.Errorf("finding config dir: %w", err)
		}
		team := ws.Team
		if team == "" {
			team = "default"
		}
		outputDir = filepath.Join(configDir, "slack-emoji-tools", "emojis", team)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	g := grabber.NewGrabber(slack.New(ws.BotToken))
	g.OutputDir = outputDir

	fmt.Printf("Downloading emoji to %s\n", outputDir)
	return g.Run(ctx)
}
```

**Step 2: Verify it compiles**

```bash
go build ./...
```

**Step 3: Commit**

```bash
git add cmd/grab.go
git commit -m "feat: add grab command"
```

---

### Task 9: Create `list` command

**Files:**
- Create: `cmd/list.go`

**Step 1: Create cmd/list.go**

```go
package cmd

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/slack-go/slack"
	"github.com/spf13/cobra"
	"github.com/stahnma/slack-emoji-tools/internal/config"
)

var listFormat string
var listIncludeAliases bool

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List custom emoji names in a workspace",
	RunE:  runList,
}

func init() {
	listCmd.Flags().StringVar(&listFormat, "format", "text", "output format: text, json, csv")
	listCmd.Flags().BoolVar(&listIncludeAliases, "include-aliases", false, "include alias emoji")
	rootCmd.AddCommand(listCmd)
}

func runList(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	ws, err := cfg.ResolveWorkspace(workspace)
	if err != nil {
		return err
	}
	if ws.BotToken == "" {
		return fmt.Errorf("bot token required for list — set bot_token in config or SLACK_TOKEN env var")
	}

	client := slack.New(ws.BotToken)
	emojis, err := client.GetEmoji()
	if err != nil {
		return fmt.Errorf("fetching emoji list: %w", err)
	}

	type emojiEntry struct {
		Name  string `json:"name"`
		Alias string `json:"alias,omitempty"`
	}

	var entries []emojiEntry
	for name, url := range emojis {
		if strings.HasPrefix(url, "alias:") {
			if listIncludeAliases {
				entries = append(entries, emojiEntry{Name: name, Alias: strings.TrimPrefix(url, "alias:")})
			}
		} else {
			entries = append(entries, emojiEntry{Name: name})
		}
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name < entries[j].Name })

	switch listFormat {
	case "json":
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(entries)
	case "csv":
		w := csv.NewWriter(os.Stdout)
		w.Write([]string{"name", "alias"})
		for _, e := range entries {
			w.Write([]string{e.Name, e.Alias})
		}
		w.Flush()
		return w.Error()
	default:
		for _, e := range entries {
			if e.Alias != "" {
				fmt.Printf("%s → %s\n", e.Name, e.Alias)
			} else {
				fmt.Println(e.Name)
			}
		}
	}
	return nil
}
```

**Step 2: Verify it compiles**

```bash
go build ./...
```

**Step 3: Commit**

```bash
git add cmd/list.go
git commit -m "feat: add list command with text/json/csv output"
```

---

### Task 10: Create `upload` command

**Files:**
- Create: `cmd/upload.go`

**Step 1: Create cmd/upload.go**

Port the upload logic from `slack-emoji-uploader/cmd/upload.go`. Key changes:
- Use `config.Load()` + `ResolveWorkspace()` for auth instead of `resolveAuth()`
- Use `state.StateDir(team)` for state file paths
- Use `config.StripCookiePrefix()` for cookie handling
- Use `uploader.FetchToken()` for token derivation
- Use slog for verbose output

**Step 2: Verify it compiles**

```bash
go build ./...
```

**Step 3: Commit**

```bash
git add cmd/upload.go
git commit -m "feat: add upload command with state tracking and conflict detection"
```

---

### Task 11: Create `status` command

**Files:**
- Create: `cmd/status.go`

**Step 1: Create cmd/status.go**

Port from `slack-emoji-uploader/cmd/status.go`. Key changes:
- Use config/workspace resolution for the team name
- Use `state.StateDir(team)` for state paths
- Accept `<directory>` arg to count total files

**Step 2: Verify it compiles**

```bash
go build ./...
```

**Step 3: Commit**

```bash
git add cmd/status.go
git commit -m "feat: add status command"
```

---

### Task 12: Create `resolve` command

**Files:**
- Create: `cmd/resolve.go`

**Step 1: Create cmd/resolve.go**

Port from `slack-emoji-uploader/cmd/resolve.go`. Same auth/state changes as upload.

**Step 2: Verify it compiles**

```bash
go build ./...
```

**Step 3: Commit**

```bash
git add cmd/resolve.go
git commit -m "feat: add resolve command for interactive conflict resolution"
```

---

### Task 13: Create Makefile

**Files:**
- Create: `Makefile`

**Step 1: Create Makefile**

```makefile
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -ldflags "-X github.com/stahnma/slack-emoji-tools/cmd.version=$(VERSION)"
BINARY  := slack-emoji-tools

.PHONY: build install test lint tidy clean

build:
	CGO_ENABLED=0 go build $(LDFLAGS) -o $(BINARY) .

install:
	CGO_ENABLED=0 go install $(LDFLAGS) .

test:
	go test ./... -v

lint:
	golangci-lint run ./...

tidy:
	go mod tidy

clean:
	rm -f $(BINARY)
```

**Step 2: Verify build**

```bash
make build
./slack-emoji-tools --help
```

Expected: help output showing all subcommands.

**Step 3: Run all tests**

```bash
make test
```

Expected: all tests pass.

**Step 4: Commit**

```bash
git add Makefile
git commit -m "feat: add Makefile with standard targets"
```

---

### Task 14: Add version command

**Files:**
- Modify: `cmd/root.go`

**Step 1: Add version subcommand to root.go**

Add a `version` command that prints the version variable (set via ldflags).

```go
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(version)
	},
}

// In init():
rootCmd.AddCommand(versionCmd)
```

**Step 2: Verify**

```bash
go build ./...
./slack-emoji-tools version
```

Expected: prints "dev"

**Step 3: Commit**

```bash
git add cmd/root.go
git commit -m "feat: add version command"
```

---

### Task 15: Final integration check

**Step 1: Run full test suite**

```bash
go test ./... -v -count=1
```

Expected: all tests pass.

**Step 2: Build and verify CLI**

```bash
make build
./slack-emoji-tools --help
./slack-emoji-tools grab --help
./slack-emoji-tools list --help
./slack-emoji-tools upload --help
./slack-emoji-tools status --help
./slack-emoji-tools resolve --help
./slack-emoji-tools version
```

Expected: all commands show correct help text.

**Step 3: Run go vet**

```bash
go vet ./...
```

Expected: no issues.

**Step 4: Commit any fixes if needed, then tidy**

```bash
go mod tidy
git add go.mod go.sum
git commit -m "chore: go mod tidy"
```
