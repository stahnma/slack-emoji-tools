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

var grabCmd = &cobra.Command{
	Use:   "grab",
	Short: "Download all custom emoji from a Slack workspace",
	RunE:  runGrab,
}

func init() {
	grabCmd.Flags().String("output", "", "output directory for downloaded emoji")
	grabCmd.Flags().Bool("skip-aliases", true, "skip alias emoji")
	rootCmd.AddCommand(grabCmd)
}

func runGrab(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	ws, err := cfg.ResolveWorkspace(workspace)
	if err != nil {
		return err
	}

	if ws.BotToken == "" {
		return fmt.Errorf("bot_token is required for the grab command")
	}

	// Resolve output directory: flag > config defaults > default path
	outputDir, _ := cmd.Flags().GetString("output")
	if outputDir == "" {
		outputDir = cfg.Defaults.OutputDir
	}
	if outputDir == "" {
		configDir, err := os.UserConfigDir()
		if err != nil {
			return fmt.Errorf("determining config directory: %w", err)
		}
		team := ws.Team
		if team == "" {
			team = "default"
		}
		outputDir = filepath.Join(configDir, "slack-emoji-tools", "emojis", team)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Downloading emoji to %s\n", outputDir)

	skipAliases, _ := cmd.Flags().GetBool("skip-aliases")

	client := slack.New(ws.BotToken)
	g := grabber.NewGrabber(client)
	g.OutputDir = outputDir
	g.SkipAliases = skipAliases

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	return g.Run(ctx)
}
