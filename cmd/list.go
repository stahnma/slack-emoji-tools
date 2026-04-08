package cmd

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/slack-go/slack"
	"github.com/spf13/cobra"
	"github.com/stahnma/slack-emoji-tools/internal/config"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List custom emoji in a Slack workspace",
	RunE:  runList,
}

func init() {
	listCmd.Flags().String("format", "text", "output format: text, json, or csv")
	listCmd.Flags().Bool("include-aliases", false, "include alias emoji in output")
	rootCmd.AddCommand(listCmd)
}

type emojiEntry struct {
	Name  string `json:"name"`
	Alias string `json:"alias,omitempty"`
}

func runList(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	ws, err := cfg.ResolveWorkspace(workspace)
	if err != nil {
		return err
	}

	if ws.BotToken == "" {
		return fmt.Errorf("bot_token is required for the list command")
	}

	format, _ := cmd.Flags().GetString("format")
	includeAliases, _ := cmd.Flags().GetBool("include-aliases")

	client := slack.New(ws.BotToken)
	emojis, err := client.GetEmoji()
	if err != nil {
		return fmt.Errorf("fetching emoji list: %w", err)
	}

	var entries []emojiEntry
	for name, uri := range emojis {
		if strings.HasPrefix(uri, "alias:") {
			if !includeAliases {
				continue
			}
			target := strings.TrimPrefix(uri, "alias:")
			entries = append(entries, emojiEntry{Name: name, Alias: target})
		} else {
			entries = append(entries, emojiEntry{Name: name})
		}
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name < entries[j].Name
	})

	out := cmd.OutOrStdout()

	switch format {
	case "text":
		for _, e := range entries {
			if e.Alias != "" {
				fmt.Fprintf(out, "%s → %s\n", e.Name, e.Alias)
			} else {
				fmt.Fprintln(out, e.Name)
			}
		}
	case "json":
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		if err := enc.Encode(entries); err != nil {
			return fmt.Errorf("encoding JSON: %w", err)
		}
	case "csv":
		fmt.Fprintln(out, "name,alias")
		for _, e := range entries {
			fmt.Fprintf(out, "%s,%s\n", e.Name, e.Alias)
		}
	default:
		return fmt.Errorf("unknown format %q: use text, json, or csv", format)
	}

	return nil
}
