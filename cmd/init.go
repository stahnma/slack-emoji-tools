package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var updateCookie bool

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Interactive setup wizard to create or update the config file",
	RunE:  runInit,
}

func init() {
	initCmd.Flags().BoolVar(&updateCookie, "update-cookie", false, "fast path to replace just the cookie for a workspace")
	rootCmd.AddCommand(initCmd)
}

// configData mirrors the YAML config structure for marshalling.
type configData struct {
	DefaultWorkspace string                        `yaml:"default_workspace"`
	Workspaces       map[string]workspaceData      `yaml:"workspaces"`
	Defaults         defaultsData                  `yaml:"defaults"`
}

type workspaceData struct {
	BotToken string `yaml:"bot_token,omitempty"`
	Cookie   string `yaml:"cookie,omitempty"`
	Team     string `yaml:"team,omitempty"`
}

type defaultsData struct {
	OutputDir   string `yaml:"output_dir,omitempty"`
	UploadDelay string `yaml:"upload_delay,omitempty"`
	AutoSuffix  bool   `yaml:"auto_suffix,omitempty"`
}

func runInit(cmd *cobra.Command, args []string) error {
	reader := bufio.NewReader(os.Stdin)

	configPath, err := defaultConfigPath()
	if err != nil {
		return err
	}

	// Load existing config if present.
	existing := loadExistingConfig(configPath)

	if updateCookie {
		return runUpdateCookie(reader, existing, configPath)
	}

	// If existing config has workspaces, offer to add or edit.
	if len(existing.Workspaces) > 0 {
		fmt.Printf("Existing config found at %s\n", configPath)
		fmt.Printf("Configured workspaces: %s\n", strings.Join(workspaceNames(existing), ", "))
		fmt.Println()
	} else {
		fmt.Println("Welcome to slack-emoji-tools setup!")
		fmt.Println()
	}

	// Loop to add workspaces.
	for {
		ws, name, err := promptWorkspace(reader, existing)
		if err != nil {
			return err
		}

		if existing.Workspaces == nil {
			existing.Workspaces = make(map[string]workspaceData)
		}
		existing.Workspaces[name] = ws

		// Set default workspace if this is the first one.
		if existing.DefaultWorkspace == "" {
			existing.DefaultWorkspace = name
		}

		// Prompt for defaults only on first workspace added this session,
		// and only if defaults are not already set.
		if existing.Defaults.OutputDir == "" && existing.Defaults.UploadDelay == "" {
			fmt.Println()
			fmt.Println("── Defaults ──")
			outputDir := prompt(reader, "Output directory for grab [./emojis]: ")
			if outputDir == "" {
				outputDir = "./emojis"
			}
			existing.Defaults.OutputDir = outputDir

			delay := prompt(reader, "Upload delay [1s]: ")
			if delay == "" {
				delay = "1s"
			}
			existing.Defaults.UploadDelay = delay
		}

		if err := writeConfig(configPath, existing); err != nil {
			return err
		}
		fmt.Printf("\nConfig written to %s\n\n", configPath)

		another := prompt(reader, "Add another workspace? (y/n): ")
		if !strings.EqualFold(strings.TrimSpace(another), "y") {
			break
		}
		fmt.Println()
	}

	fmt.Println("\nYou're all set! Try: slack-emoji-tools list")
	return nil
}

func runUpdateCookie(reader *bufio.Reader, cfg configData, configPath string) error {
	if len(cfg.Workspaces) == 0 {
		return fmt.Errorf("no workspaces configured; run 'slack-emoji-tools init' first")
	}

	// Determine which workspace to update.
	wsName := workspace // from --workspace flag
	if wsName == "" {
		wsName = cfg.DefaultWorkspace
	}
	if wsName == "" {
		fmt.Printf("Available workspaces: %s\n", strings.Join(workspaceNames(cfg), ", "))
		wsName = prompt(reader, "Workspace to update: ")
		wsName = strings.TrimSpace(wsName)
	}

	ws, ok := cfg.Workspaces[wsName]
	if !ok {
		return fmt.Errorf("workspace %q not found in config", wsName)
	}

	fmt.Printf("Updating cookie for workspace %q\n", wsName)
	cookie := prompt(reader, "Browser cookie (xoxd-...): ")
	cookie = strings.TrimSpace(cookie)

	if cookie != "" && !strings.HasPrefix(cookie, "xoxd-") {
		fmt.Fprintln(os.Stderr, "Warning: cookie does not start with 'xoxd-' — saving anyway")
	}

	ws.Cookie = cookie
	cfg.Workspaces[wsName] = ws

	if err := writeConfig(configPath, cfg); err != nil {
		return err
	}
	fmt.Printf("Config updated at %s\n", configPath)
	return nil
}

func promptWorkspace(reader *bufio.Reader, existing configData) (workspaceData, string, error) {
	name := prompt(reader, "Workspace name (e.g. mycompany): ")
	name = strings.TrimSpace(name)
	if name == "" {
		return workspaceData{}, "", fmt.Errorf("workspace name is required")
	}

	// If workspace already exists, show that we're updating it.
	if _, ok := existing.Workspaces[name]; ok {
		fmt.Printf("Updating existing workspace %q\n", name)
	}

	fmt.Println()
	fmt.Println("── Grab/List credentials (optional) ──")
	fmt.Println("These use a Slack Bot Token with emoji:read scope.")
	botToken := prompt(reader, "Bot token (xoxb-...): ")
	botToken = strings.TrimSpace(botToken)

	if botToken != "" && !strings.HasPrefix(botToken, "xoxb-") {
		fmt.Fprintln(os.Stderr, "Warning: bot token does not start with 'xoxb-' — saving anyway")
	}

	fmt.Println()
	fmt.Println("── Upload credentials (optional) ──")
	fmt.Println("These use a browser session cookie from Slack.")
	fmt.Println("See: https://github.com/stahnma/slack-emoji-tools#getting-credentials")
	cookie := prompt(reader, "Browser cookie (xoxd-...): ")
	cookie = strings.TrimSpace(cookie)

	if cookie != "" && !strings.HasPrefix(cookie, "xoxd-") {
		fmt.Fprintln(os.Stderr, "Warning: cookie does not start with 'xoxd-' — saving anyway")
	}

	teamDefault := name
	teamPrompt := fmt.Sprintf("Team subdomain [%s]: ", teamDefault)
	team := prompt(reader, teamPrompt)
	team = strings.TrimSpace(team)
	if team == "" {
		team = teamDefault
	}

	// Mask credentials in confirmation output.
	fmt.Println()
	if botToken != "" {
		fmt.Printf("  Bot token: %s\n", maskValue(botToken))
	}
	if cookie != "" {
		fmt.Printf("  Cookie:    %s\n", maskValue(cookie))
	}
	fmt.Printf("  Team:      %s\n", team)

	return workspaceData{
		BotToken: botToken,
		Cookie:   cookie,
		Team:     team,
	}, name, nil
}

func prompt(reader *bufio.Reader, message string) string {
	fmt.Print(message)
	line, _ := reader.ReadString('\n')
	return strings.TrimRight(line, "\n\r")
}

func maskValue(s string) string {
	if len(s) <= 8 {
		return strings.Repeat("*", len(s))
	}
	return s[:4] + "..." + s[len(s)-4:]
}

func defaultConfigPath() (string, error) {
	if cfgFile != "" {
		return cfgFile, nil
	}
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("determining config directory: %w", err)
	}
	return filepath.Join(configDir, "slack-emoji-tools", "config.yaml"), nil
}

func loadExistingConfig(path string) configData {
	data, err := os.ReadFile(path)
	if err != nil {
		return configData{}
	}
	var cfg configData
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return configData{}
	}
	return cfg
}

func writeConfig(path string, cfg configData) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	data, err := yaml.Marshal(&cfg)
	if err != nil {
		return fmt.Errorf("marshalling config: %w", err)
	}

	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("writing config file: %w", err)
	}
	return nil
}

func workspaceNames(cfg configData) []string {
	names := make([]string, 0, len(cfg.Workspaces))
	for name := range cfg.Workspaces {
		names = append(names, name)
	}
	return names
}
