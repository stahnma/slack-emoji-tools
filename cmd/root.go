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
