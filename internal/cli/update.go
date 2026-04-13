package cli

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/fieldpulse-prototypes/fpproto/internal/api"
	"github.com/fieldpulse-prototypes/fpproto/internal/auth"
	"github.com/fieldpulse-prototypes/fpproto/internal/config"
	"github.com/fieldpulse-prototypes/fpproto/internal/ui"
	"github.com/spf13/cobra"
)

var Version = "dev"

func NewUpdateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "update",
		Short: "Update fpproto to the latest version",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(ui.HeaderStyle.Render("  Update fpproto"))
			fmt.Println()

			var release *api.GitHubRelease

			steps := []ui.Step{
				{
					Title: "Checking for updates",
					Action: func() (string, error) {
						_, token, err := auth.EnsureAuth()
						if err != nil {
							return "", fmt.Errorf("auth failed: %w", err)
						}

						github := api.NewGitHubClient(token, config.RemoteOrg)

						r, err := github.GetLatestRelease(config.CLIRepo)
						if err != nil {
							return "", fmt.Errorf("failed to check for updates: %w", err)
						}

						if r == nil {
							fmt.Println("\n  Already on the latest version")
							return "already on latest", nil
						}

						if r.TagName == Version {
							fmt.Println("\n  Already on the latest version")
							return "already on latest", nil
						}

						release = r
						return fmt.Sprintf("%s available (current: %s)", release.TagName, Version), nil
					},
				},
				{
					Title: "Downloading update",
					Action: func() (string, error) {
						if release == nil {
							return "skipped", nil
						}

						binaryName := fmt.Sprintf("fpproto-%s-%s", runtime.GOOS, runtime.GOARCH)

						var downloadURL string
						for _, asset := range release.Assets {
							if asset.Name == binaryName {
								downloadURL = asset.BrowserDownloadURL
								break
							}
						}

						if downloadURL == "" {
							return "", fmt.Errorf("no binary found for %s/%s", runtime.GOOS, runtime.GOARCH)
						}

						resp, err := http.Get(downloadURL)
						if err != nil {
							return "", fmt.Errorf("failed to download update: %w", err)
						}
						defer resp.Body.Close()

						if resp.StatusCode != http.StatusOK {
							return "", fmt.Errorf("download failed with status %d", resp.StatusCode)
						}

						tmpFile, err := os.CreateTemp("", "fpproto-update-*")
						if err != nil {
							return "", fmt.Errorf("failed to create temp file: %w", err)
						}
						defer tmpFile.Close()

						if _, err := io.Copy(tmpFile, resp.Body); err != nil {
							os.Remove(tmpFile.Name())
							return "", fmt.Errorf("failed to write update: %w", err)
						}
						tmpFile.Close()

						if err := os.Chmod(tmpFile.Name(), 0755); err != nil {
							os.Remove(tmpFile.Name())
							return "", fmt.Errorf("failed to set permissions: %w", err)
						}

						execPath, err := os.Executable()
						if err != nil {
							os.Remove(tmpFile.Name())
							return "", fmt.Errorf("failed to find current executable: %w", err)
						}

						execPath, err = filepath.EvalSymlinks(execPath)
						if err != nil {
							os.Remove(tmpFile.Name())
							return "", fmt.Errorf("failed to resolve executable path: %w", err)
						}

						if err := os.Rename(tmpFile.Name(), execPath); err != nil {
							os.Remove(tmpFile.Name())
							return "", fmt.Errorf("failed to replace binary: %w", err)
						}

						return fmt.Sprintf("updated to %s", release.TagName), nil
					},
				},
			}

			if err := ui.RunSteps(steps); err != nil {
				fmt.Fprintf(os.Stderr, "\n  %s %s\n", ui.ErrorIcon, err)
				os.Exit(1)
			}

			if release != nil && release.Body != "" {
				fmt.Println()
				fmt.Println(ui.MutedStyle.Render("  Changelog:"))
				fmt.Println(ui.MutedStyle.Render("  " + release.Body))
			}

			fmt.Println()
		},
	}
}

// CheckForUpdate checks for a newer version in the background.
// It returns a styled notification string if an update is available, or "" otherwise.
// Safe to call from a background goroutine — all errors are silently swallowed.
func CheckForUpdate(currentVersion string) string {
	checkFile := filepath.Join(config.ConfigPath(), "last_update_check")

	if data, err := os.ReadFile(checkFile); err == nil {
		if ts, err := strconv.ParseInt(string(data), 10, 64); err == nil {
			lastCheck := time.Unix(ts, 0)
			if time.Since(lastCheck) < time.Hour {
				return ""
			}
		}
	}

	_, token, err := auth.EnsureAuth()
	if err != nil || token == "" {
		return ""
	}

	github := api.NewGitHubClient(token, config.RemoteOrg)

	release, err := github.GetLatestRelease(config.CLIRepo)
	if err != nil || release == nil {
		writeUpdateTimestamp(checkFile)
		return ""
	}

	if release.TagName == currentVersion {
		writeUpdateTimestamp(checkFile)
		return ""
	}

	writeUpdateTimestamp(checkFile)

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("39")).
		Padding(0, 2)

	content := fmt.Sprintf(
		"Update available: %s %s %s\nRun %s to upgrade",
		ui.MutedStyle.Render(currentVersion),
		"\u2192",
		ui.AccentStyle.Render(release.TagName),
		ui.AccentStyle.Render("fpproto update"),
	)

	return boxStyle.Render(content)
}

func writeUpdateTimestamp(path string) {
	dir := filepath.Dir(path)
	os.MkdirAll(dir, 0755)
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	os.WriteFile(path, []byte(ts), 0644)
}
