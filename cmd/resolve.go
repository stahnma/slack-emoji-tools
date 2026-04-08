package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/stahnma/slack-emoji-tools/internal/config"
	"github.com/stahnma/slack-emoji-tools/internal/state"
	"github.com/stahnma/slack-emoji-tools/internal/uploader"
)

var resolveCmd = &cobra.Command{
	Use:   "resolve <directory>",
	Short: "Interactively resolve emoji name conflicts",
	Args:  cobra.ExactArgs(1),
	RunE:  runResolve,
}

func init() {
	rootCmd.AddCommand(resolveCmd)
}

func runResolve(cmd *cobra.Command, args []string) error {
	dir := args[0]

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	ws, err := cfg.ResolveWorkspace(workspace)
	if err != nil {
		return err
	}
	cookie := config.StripCookiePrefix(ws.Cookie)
	team := ws.Team

	if cookie == "" || team == "" {
		return fmt.Errorf("cookie and team are required — set them in config or via environment variables")
	}

	stateDir, err := state.StateDir(team)
	if err != nil {
		return fmt.Errorf("determining state directory: %w", err)
	}

	st, err := state.Load(stateDir)
	if err != nil {
		return fmt.Errorf("loading state: %w", err)
	}
	conflicts, err := state.LoadConflicts(stateDir)
	if err != nil {
		return fmt.Errorf("loading conflicts: %w", err)
	}

	if len(conflicts.Entries) == 0 {
		fmt.Println("No conflicts to resolve.")
		return nil
	}

	fmt.Print("Fetching API token... ")
	token, err := uploader.FetchToken(cookie, team)
	if err != nil {
		return fmt.Errorf("fetching token: %w", err)
	}
	fmt.Println("ok")

	client := uploader.NewClient(token, cookie, team, 1*time.Second)
	client.Verbose = verbose
	reader := bufio.NewReader(os.Stdin)
	resolved := 0

	for filePath, entry := range conflicts.Entries {
		fmt.Printf("\nConflict: %s → :%s: already exists\n", filePath, entry.Name)
		fmt.Printf("Attempted: %s\n", strings.Join(entry.Attempted, ", "))
		fmt.Print("Enter new name (or 'skip'): ")

		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		if input == "" || input == "skip" {
			fmt.Println("Skipped.")
			continue
		}

		imageData, err := os.ReadFile(filepath.Join(dir, filePath))
		if err != nil {
			fmt.Printf("Error reading file: %v\n", err)
			continue
		}

		fmt.Printf("Uploading: %s → :%s: ... ", filePath, input)
		result, err := client.UploadEmoji(input, imageData, filepath.Base(filePath))
		if err != nil {
			fmt.Printf("error: %v\n", err)
			continue
		}

		if result.OK {
			fmt.Println("OK")
			st.RecordSuccess(filePath, input)
			conflicts.Remove(filePath)
			resolved++
		} else {
			fmt.Printf("failed: %s\n", result.Error)
			entry.Attempted = append(entry.Attempted, input)
			entry.LastAttempt = time.Now().UTC()
			conflicts.Entries[filePath] = entry
		}

		if err := st.Save(); err != nil {
			return fmt.Errorf("saving state: %w", err)
		}
		if err := conflicts.Save(); err != nil {
			return fmt.Errorf("saving conflicts: %w", err)
		}
		time.Sleep(client.Delay())
	}

	fmt.Printf("\nResolved: %d, Remaining: %d\n", resolved, len(conflicts.Entries))
	return nil
}
