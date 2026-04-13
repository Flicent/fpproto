package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/fieldpulse-prototypes/fpproto/internal/api"
	"github.com/fieldpulse-prototypes/fpproto/internal/auth"
	"github.com/fieldpulse-prototypes/fpproto/internal/config"
	"github.com/fieldpulse-prototypes/fpproto/internal/ui"
	"github.com/spf13/cobra"
)

func NewDestroyCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "destroy <name>",
		Short: "Archive and tear down a prototype",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			name := args[0]

			fmt.Println(ui.HeaderStyle.Render("  Destroy Prototype"))
			fmt.Println()

			email, _, err := auth.EnsureAuth()
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
			supabase := api.NewSupabaseClient(cfg.SupabaseAccessToken, cfg.SupabaseOrgID)
			vercel := api.NewVercelClient(cfg.VercelToken, cfg.VercelTeamID)

			repo, err := github.GetRepo(name)
			if err != nil {
				fmt.Fprintf(os.Stderr, "  %s Failed to check repository: %s\n", ui.ErrorIcon, err)
				os.Exit(1)
			}
			if repo == nil {
				fmt.Fprintf(os.Stderr, "  %s Repository %s not found\n", ui.ErrorIcon, name)
				os.Exit(1)
			}
			if repo.Archived {
				fmt.Fprintf(os.Stderr, "  %s Repository %s is already archived\n", ui.ErrorIcon, name)
				os.Exit(1)
			}

			confirmed, err := ui.ConfirmDestroy(name)
			if err != nil {
				fmt.Fprintf(os.Stderr, "  %s %s\n", ui.ErrorIcon, err)
				os.Exit(1)
			}
			if !confirmed {
				os.Exit(0)
			}

			date := time.Now().Format("2006-01-02")

			steps := []ui.Step{
				{
					Title: "Updating README with archive banner",
					Action: func() (string, error) {
						content, sha, err := github.GetRepoContents(name, "README.md")
						if err != nil {
							return "", fmt.Errorf("failed to fetch README: %w", err)
						}

						banner := fmt.Sprintf("> **ARCHIVED** \u2014 This prototype was archived on %s by %s.\n\n", date, email)
						newContent := append([]byte(banner), content...)

						msg := "Archive: prototype archived by " + email
						if err := github.CreateOrUpdateFile(name, "README.md", msg, newContent, sha); err != nil {
							return "", fmt.Errorf("failed to update README: %w", err)
						}

						return "README updated", nil
					},
				},
				{
					Title: "Archiving GitHub repository",
					Action: func() (string, error) {
						if err := github.ArchiveRepo(name); err != nil {
							return "", fmt.Errorf("failed to archive repository: %w", err)
						}
						return "repository archived", nil
					},
				},
				{
					Title: "Deleting Supabase project",
					Action: func() (string, error) {
						metaBytes, _, err := github.GetRepoContents(name, ".fpproto.json")
						if err != nil {
							return "", fmt.Errorf("failed to read .fpproto.json: %w", err)
						}

						var metadata config.PrototypeMetadata
						if err := json.Unmarshal(metaBytes, &metadata); err != nil {
							return "", fmt.Errorf("failed to parse .fpproto.json: %w", err)
						}

						if metadata.SupabaseProjectRef == "" {
							return "already deleted", nil
						}

						if err := supabase.DeleteProject(metadata.SupabaseProjectRef); err != nil {
							return "", fmt.Errorf("failed to delete Supabase project: %w", err)
						}

						return "project deleted", nil
					},
				},
				{
					Title: "Deleting Vercel project",
					Action: func() (string, error) {
						vercelProject, err := vercel.GetProject(name)
						if err != nil {
							return "", fmt.Errorf("failed to check Vercel project: %w", err)
						}

						if vercelProject != nil {
							if err := vercel.DeleteProject(vercelProject.ID); err != nil {
								return "", fmt.Errorf("failed to delete Vercel project: %w", err)
							}
							return "deployment deleted", nil
						}

						return "no deployment found", nil
					},
				},
			}

			if err := ui.RunSteps(steps); err != nil {
				fmt.Fprintf(os.Stderr, "\n  %s %s\n", ui.ErrorIcon, err)
				os.Exit(1)
			}

			fmt.Println("\n  " + ui.AccentStyle.Render(name) + " has been archived.")
			fmt.Println()
		},
	}
}
