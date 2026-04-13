package cli

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"sort"
	"time"

	"github.com/fieldpulse-prototypes/fpproto/internal/api"
	"github.com/fieldpulse-prototypes/fpproto/internal/auth"
	"github.com/fieldpulse-prototypes/fpproto/internal/config"
	"github.com/fieldpulse-prototypes/fpproto/internal/ui"
	"github.com/spf13/cobra"
)

type prototypeEntry struct {
	name         string
	createdBy    string
	createdAt    time.Time
	url          string
	supabaseMode string
}

func NewListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List active prototypes",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(ui.HeaderStyle.Render("  List Prototypes"))
			fmt.Println()

			_, _, err := auth.EnsureAuth()
			if err != nil {
				fmt.Fprintf(os.Stderr, "  %s %s\n", ui.ErrorIcon, err)
				os.Exit(1)
			}

			cfg, err := config.Load()
			if err != nil {
				fmt.Fprintf(os.Stderr, "  %s %s\n", ui.ErrorIcon, err)
				os.Exit(1)
			}

			github := api.NewGitHubClient(cfg.SupabaseAccessToken, config.RemoteOrg)
			vercel := api.NewVercelClient(cfg.VercelToken, cfg.VercelTeamID)

			repos, err := github.ListOrgRepos()
			if err != nil {
				fmt.Fprintf(os.Stderr, "  %s Failed to list repositories: %s\n", ui.ErrorIcon, err)
				os.Exit(1)
			}

			excluded := map[string]bool{
				config.RemoteConfigRepo: true,
				config.TemplateRepo:     true,
				config.CLIRepo:          true,
			}

			var entries []prototypeEntry

			for _, repo := range repos {
				if repo.Archived {
					continue
				}
				if excluded[repo.Name] {
					continue
				}

				metaBytes, _, err := github.GetRepoContents(repo.Name, ".fpproto.json")
				if err != nil {
					continue
				}

				var metadata config.PrototypeMetadata
				if err := json.Unmarshal(metaBytes, &metadata); err != nil {
					continue
				}

				var url string
				vercelProject, err := vercel.GetProject(repo.Name)
				if err == nil && vercelProject != nil {
					url = vercelProject.URL()
				}

				createdAt, _ := time.Parse(time.RFC3339, metadata.CreatedAt)

				mode := metadata.SupabaseMode
				if mode == "" {
					mode = config.SupabaseModeLive
				}

				entries = append(entries, prototypeEntry{
					name:         repo.Name,
					createdBy:    metadata.CreatedBy,
					createdAt:    createdAt,
					url:          url,
					supabaseMode: mode,
				})
			}

			sort.Slice(entries, func(i, j int) bool {
				return entries[i].createdAt.After(entries[j].createdAt)
			})

			if len(entries) == 0 {
				fmt.Println("  No active prototypes. Run `fpproto create <name>` to start one.")
				fmt.Println()
				return
			}

			fmt.Printf("\n  Active Prototypes (%d)\n\n", len(entries))

			headers := []string{"NAME", "CREATED BY", "CREATED", "SUPABASE", "URL"}
			rows := make([][]string, 0, len(entries))

			for _, e := range entries {
				modeLabel := e.supabaseMode
				if modeLabel == config.SupabaseModeLocal {
					modeLabel = ui.MutedStyle.Render("local")
				} else {
					modeLabel = ui.WarningStyle.Render("live")
				}
				rows = append(rows, []string{
					ui.AccentStyle.Render(e.name),
					e.createdBy,
					relativeTime(e.createdAt),
					modeLabel,
					ui.URLStyle.Render(e.url),
				})
			}

			fmt.Println(ui.RenderTable(headers, rows))
		},
	}
}

func relativeTime(t time.Time) string {
	if t.IsZero() {
		return "unknown"
	}

	now := time.Now()
	d := now.Sub(t)

	if d < time.Minute {
		return "just now"
	}

	minutes := int(math.Floor(d.Minutes()))
	if minutes < 60 {
		if minutes == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", minutes)
	}

	hours := int(math.Floor(d.Hours()))
	if hours < 24 {
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	}

	days := int(math.Floor(float64(hours) / 24))
	if days < 7 {
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	}

	weeks := int(math.Floor(float64(days) / 7))
	if weeks < 5 {
		if weeks == 1 {
			return "1 week ago"
		}
		return fmt.Sprintf("%d weeks ago", weeks)
	}

	months := int(math.Floor(float64(days) / 30))
	if months == 1 {
		return "1 month ago"
	}
	return fmt.Sprintf("%d months ago", months)
}
