package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/fieldpulse-prototypes/fpproto/internal/api"
	"github.com/fieldpulse-prototypes/fpproto/internal/auth"
	"github.com/fieldpulse-prototypes/fpproto/internal/config"
	"github.com/fieldpulse-prototypes/fpproto/internal/ui"
	"github.com/spf13/cobra"
)

func NewSetupCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "setup",
		Short: "Configure fpproto with org credentials",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(ui.HeaderStyle.Render("fpproto setup"))
			fmt.Println()

			var email, token string
			var cfg *config.Config

			err := ui.RunSteps([]ui.Step{
				{
					Title: "Checking GitHub CLI",
					Action: func() (string, error) {
						var err error
						email, token, err = auth.EnsureAuth()
						if err != nil {
							return "", err
						}
						return email, nil
					},
				},
				{
					Title: "Pulling org config",
					Action: func() (string, error) {
						gh := api.NewGitHubClient(token, config.RemoteOrg)
						contents, _, err := gh.GetRepoContents(config.RemoteConfigRepo, config.RemoteConfigPath)
						if err != nil {
							return "", fmt.Errorf("failed to fetch org config: %w", err)
						}

						var remoteConfig config.RemoteConfig
						if err := json.Unmarshal(contents, &remoteConfig); err != nil {
							return "", fmt.Errorf("failed to parse org config: %w", err)
						}

						cfg = &config.Config{
							SupabaseAccessToken: remoteConfig.SupabaseAccessToken,
							SupabaseOrgID:       remoteConfig.SupabaseOrgID,
							VercelToken:         remoteConfig.VercelToken,
							VercelTeamID:        remoteConfig.VercelTeamID,
							ConfigVersion:       remoteConfig.ConfigVersion,
							UserEmail:           email,
							SupabaseDeployHash:  remoteConfig.SupabaseDeployHash,
						}

						if err := config.Save(cfg); err != nil {
							return "", fmt.Errorf("failed to save config: %w", err)
						}

						return fmt.Sprintf("v%d", remoteConfig.ConfigVersion), nil
					},
				},
				{
					Title: "Validating Supabase connection",
					Action: func() (string, error) {
						supabase := api.NewSupabaseClient(cfg.SupabaseAccessToken, cfg.SupabaseOrgID)
						if _, err := supabase.ListProjects(); err != nil {
							return "", fmt.Errorf("supabase connection failed: %w", err)
						}
						return "connected", nil
					},
				},
				{
					Title: "Validating Vercel connection",
					Action: func() (string, error) {
						vercel := api.NewVercelClient(cfg.VercelToken, cfg.VercelTeamID)
						if err := vercel.GetTeam(); err != nil {
							return "", fmt.Errorf("vercel connection failed: %w", err)
						}
						return "connected", nil
					},
				},
			})

			if err != nil {
				os.Exit(1)
			}

			fmt.Print("\n  You're all set! Run fpproto create <name> to start a prototype.\n\n")
		},
	}
}
