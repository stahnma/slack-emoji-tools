package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/slack-go/slack"
	"github.com/spf13/cobra"
	"github.com/stahnma/slack-emoji-tools/internal/config"
	"github.com/stahnma/slack-emoji-tools/internal/state"
	"github.com/stahnma/slack-emoji-tools/internal/uploader"
	"github.com/stahnma/slack-emoji-tools/internal/walker"
)

var (
	flagAutoSuffix bool
	flagDelay      time.Duration
	flagDryRun     bool
)

var uploadCmd = &cobra.Command{
	Use:           "upload <directory>",
	Short:         "Upload emoji from a directory to Slack",
	Args:          cobra.ExactArgs(1),
	RunE:          runUpload,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	uploadCmd.Flags().BoolVar(&flagAutoSuffix, "auto-suffix", false, "Automatically append numeric suffix on name conflicts")
	uploadCmd.Flags().DurationVar(&flagDelay, "delay", 1*time.Second, "Delay between uploads")
	uploadCmd.Flags().BoolVar(&flagDryRun, "dry-run", false, "Show what would be uploaded without uploading")
	rootCmd.AddCommand(uploadCmd)
}

func runUpload(cmd *cobra.Command, args []string) error {
	dir := args[0]

	entries, err := walker.WalkDir(dir)
	if err != nil {
		return fmt.Errorf("walking directory: %w", err)
	}
	if len(entries) == 0 {
		fmt.Println("No emoji files found.")
		return nil
	}

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

	if flagDryRun {
		for _, entry := range entries {
			if st.IsUploaded(entry.Path) {
				continue
			}
			if _, hasConflict := conflicts.Entries[entry.Path]; hasConflict {
				continue
			}
			fmt.Printf("[dry-run] Would upload: %s → :%s:\n", entry.Path, entry.Name)
		}
		return nil
	}

	fmt.Print("Fetching API token... ")
	token, err := uploader.FetchToken(cookie, team)
	if err != nil {
		return fmt.Errorf("fetching token: %w", err)
	}
	fmt.Println("ok")

	client := uploader.NewClient(token, cookie, team, flagDelay)
	client.Verbose = verbose

	// Set up dedup support: fetch emoji list on first conflict if bot_token is available.
	botToken := ws.BotToken
	var emojiMap map[string]string // lazily populated on first conflict
	var emojiMapFetched bool
	httpClient := &http.Client{Timeout: 30 * time.Second}

	fetchEmojiMap := func() map[string]string {
		if emojiMapFetched {
			return emojiMap
		}
		emojiMapFetched = true
		if botToken == "" {
			slog.Info("no bot_token configured; skipping SHA256 duplicate detection")
			return nil
		}
		slog.Info("fetching emoji list for duplicate detection")
		api := slack.New(botToken)
		em, err := api.GetEmoji()
		if err != nil {
			slog.Warn("could not fetch emoji list for dedup", "error", err)
			return nil
		}
		emojiMap = em
		return emojiMap
	}

	var uploaded, skipped, conflicted, duplicates int

	for _, entry := range entries {
		if st.IsUploaded(entry.Path) {
			skipped++
			continue
		}
		if _, hasConflict := conflicts.Entries[entry.Path]; hasConflict {
			skipped++
			continue
		}

		name := entry.Name
		attempted := []string{}
		success := false
		isConflict := false

		imageData, err := os.ReadFile(filepath.Join(dir, entry.Path))
		if err != nil {
			return fmt.Errorf("reading %s: %w", entry.Path, err)
		}

		for suffix := 0; suffix < 100; suffix++ {
			candidateName := name
			if suffix > 0 {
				candidateName = fmt.Sprintf("%s%d", name, suffix+1)
			}
			attempted = append(attempted, candidateName)

			fmt.Printf("Uploading: %s → :%s: ... ", entry.Path, candidateName)
			result, err := client.UploadEmoji(candidateName, imageData, filepath.Base(entry.Path))
			if err != nil {
				return fmt.Errorf("uploading %s: %w", entry.Path, err)
			}

			if result.OK {
				fmt.Println("OK")
				st.RecordSuccess(entry.Path, candidateName)
				if err := st.Save(); err != nil {
					return fmt.Errorf("saving state: %w", err)
				}
				uploaded++
				success = true
				break
			}

			if result.Error == "error_name_taken" {
				// Check if the remote emoji is identical (SHA256 dedup).
				if em := fetchEmojiMap(); em != nil {
					if remoteURL, found := uploader.LookupEmojiURL(em, candidateName); found {
						isDup, dedupErr := uploader.IsDuplicate(context.Background(), httpClient, remoteURL, imageData)
						if dedupErr != nil {
							slog.Warn("dedup check failed", "emoji", candidateName, "error", dedupErr)
						} else if isDup {
							fmt.Printf("duplicate (same image: %s)\n", candidateName)
							st.RecordSuccess(entry.Path, candidateName)
							if err := st.Save(); err != nil {
								return fmt.Errorf("saving state: %w", err)
							}
							duplicates++
							success = true
							break
						}
					}
				}
				fmt.Printf("conflict (%s)\n", candidateName)
				isConflict = true
				if !flagAutoSuffix {
					break
				}
				continue
			}

			if result.Error == "ratelimited" {
				fmt.Println("rate limited (retries exhausted, will retry on next run)")
				break
			}

			if result.Error == "not_authed" || result.Error == "invalid_auth" || result.Error == "token_revoked" {
				return fmt.Errorf("authentication failed: %s — try 'slack-emoji-tools init --update-cookie' to refresh your cookie", result.Error)
			}

			fmt.Printf("error: %s (will retry on next run)\n", result.Error)
			break
		}

		if !success && isConflict {
			conflicts.RecordConflict(entry.Path, name, attempted)
			if err := conflicts.Save(); err != nil {
				return fmt.Errorf("saving conflicts: %w", err)
			}
			conflicted++
		}

		time.Sleep(client.Delay())
	}

	fmt.Printf("\nDone: %d uploaded, %d duplicates, %d skipped, %d conflicts\n", uploaded, duplicates, skipped, conflicted)
	return nil
}
