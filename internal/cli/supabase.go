package cli

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/fieldpulse-prototypes/fpproto/internal/api"
	"github.com/fieldpulse-prototypes/fpproto/internal/auth"
	"github.com/fieldpulse-prototypes/fpproto/internal/config"
	"github.com/fieldpulse-prototypes/fpproto/internal/ui"
	"github.com/spf13/cobra"
)

func NewSupabaseCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "supabase <name>",
		Short: "Add a live Supabase project to an existing prototype",
		Long:  "Upgrades a local-mode prototype to use a live Supabase cloud project. Requires the admin deploy password.",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			name := args[0]

			fmt.Println(ui.HeaderStyle.Render("fpproto supabase"))
			fmt.Println()
			fmt.Printf("  Adding live Supabase to %s...\n\n", ui.AccentStyle.Render(name))

			var token string
			var cfg *config.Config
			var metadata config.PrototypeMetadata
			var metadataSHA string
			var project *api.Project
			var anonKey, serviceRoleKey, supabaseURL string

			err := ui.RunSteps([]ui.Step{
				{
					Title: "Authenticating",
					Action: func() (string, error) {
						var err error
						_, token, err = auth.EnsureAuth()
						if err != nil {
							return "", err
						}
						return "authenticated", nil
					},
				},
				{
					Title: "Checking config",
					Action: func() (string, error) {
						var err error
						cfg, err = config.Load()
						if err != nil {
							return "", fmt.Errorf("config not found, run fpproto setup first: %w", err)
						}

						gh := api.NewGitHubClient(token, config.RemoteOrg)
						contents, _, err := gh.GetRepoContents(config.RemoteConfigRepo, config.RemoteConfigPath)
						if err == nil {
							var remoteConfig config.RemoteConfig
							if json.Unmarshal(contents, &remoteConfig) == nil {
								if remoteConfig.ConfigVersion > cfg.ConfigVersion {
									cfg.SupabaseAccessToken = remoteConfig.SupabaseAccessToken
									cfg.SupabaseOrgID = remoteConfig.SupabaseOrgID
									cfg.VercelToken = remoteConfig.VercelToken
									cfg.VercelTeamID = remoteConfig.VercelTeamID
									cfg.ConfigVersion = remoteConfig.ConfigVersion
									cfg.SupabaseDeployHash = remoteConfig.SupabaseDeployHash
									_ = config.Save(cfg)
								}
							}
						}

						return fmt.Sprintf("v%d", cfg.ConfigVersion), nil
					},
				},
				{
					Title: "Reading prototype metadata",
					Action: func() (string, error) {
						gh := api.NewGitHubClient(token, config.RemoteOrg)
						metaBytes, sha, err := gh.GetRepoContents(name, ".fpproto.json")
						if err != nil {
							return "", fmt.Errorf("failed to read .fpproto.json — is %q a valid prototype?", name)
						}
						metadataSHA = sha

						if err := json.Unmarshal(metaBytes, &metadata); err != nil {
							return "", fmt.Errorf("failed to parse .fpproto.json: %w", err)
						}

						if metadata.SupabaseMode == config.SupabaseModeLive {
							return "", fmt.Errorf("prototype %q already has a live Supabase project", name)
						}

						return "local mode confirmed", nil
					},
				},
				{
					Title: "Verifying deploy authorization",
					Action: func() (string, error) {
						if err := verifyDeployPassword(cfg); err != nil {
							return "", err
						}
						return "authorized", nil
					},
				},
				{
					Title: "Creating Supabase project",
					Action: func() (string, error) {
						randBytes := make([]byte, 16)
						if _, err := rand.Read(randBytes); err != nil {
							return "", fmt.Errorf("failed to generate password: %w", err)
						}
						dbPass := hex.EncodeToString(randBytes)

						sb := api.NewSupabaseClient(cfg.SupabaseAccessToken, cfg.SupabaseOrgID)
						var err error
						project, err = sb.CreateProject(name, dbPass, "us-east-1")
						if err != nil {
							return "", fmt.Errorf("failed to create project: %w", err)
						}

						if err := sb.WaitForProject(project.Ref, 120*time.Second); err != nil {
							return "", fmt.Errorf("project did not become active: %w", err)
						}

						return fmt.Sprintf("region: %s", project.Region), nil
					},
				},
				{
					Title: "Running migrations",
					Action: func() (string, error) {
						gh := api.NewGitHubClient(token, config.RemoteOrg)
						sql, _, err := gh.GetRepoContents(config.TemplateRepo, "supabase/migrations/001_seed_schema.sql")
						if err != nil {
							return "", fmt.Errorf("failed to fetch migration: %w", err)
						}

						sb := api.NewSupabaseClient(cfg.SupabaseAccessToken, cfg.SupabaseOrgID)
						if err := sb.RunSQL(project.Ref, string(sql)); err != nil {
							return "", fmt.Errorf("migration failed: %w", err)
						}

						return "schema created", nil
					},
				},
				{
					Title: "Loading seed data",
					Action: func() (string, error) {
						gh := api.NewGitHubClient(token, config.RemoteOrg)
						sql, _, err := gh.GetRepoContents(config.TemplateRepo, "supabase/seed.sql")
						if err != nil {
							return "", fmt.Errorf("failed to fetch seed data: %w", err)
						}

						sb := api.NewSupabaseClient(cfg.SupabaseAccessToken, cfg.SupabaseOrgID)
						if err := sb.RunSQL(project.Ref, string(sql)); err != nil {
							return "", fmt.Errorf("seed data load failed: %w", err)
						}

						return "seed data loaded", nil
					},
				},
				{
					Title: "Fetching Supabase credentials",
					Action: func() (string, error) {
						sb := api.NewSupabaseClient(cfg.SupabaseAccessToken, cfg.SupabaseOrgID)
						var err error
						anonKey, serviceRoleKey, err = sb.GetAPIKeys(project.Ref)
						if err != nil {
							return "", fmt.Errorf("failed to fetch API keys: %w", err)
						}
						supabaseURL = fmt.Sprintf("https://%s.supabase.co", project.Ref)
						return "keys fetched", nil
					},
				},
				{
					Title: "Updating Vercel environment",
					Action: func() (string, error) {
						vercel := api.NewVercelClient(cfg.VercelToken, cfg.VercelTeamID)
						vp, err := vercel.GetProject(name)
						if err != nil {
							return "", fmt.Errorf("failed to look up Vercel project: %w", err)
						}
						if vp == nil {
							return "no Vercel project found, skipped", nil
						}

						envVars := map[string]string{
							"NEXT_PUBLIC_SUPABASE_URL":      supabaseURL,
							"NEXT_PUBLIC_SUPABASE_ANON_KEY": anonKey,
							"SUPABASE_SERVICE_ROLE_KEY":     serviceRoleKey,
						}

						if err := vercel.SetEnvVars(vp.ID, envVars); err != nil {
							return "", fmt.Errorf("failed to set env vars: %w", err)
						}

						return "env vars updated", nil
					},
				},
				{
					Title: "Updating prototype metadata",
					Action: func() (string, error) {
						metadata.SupabaseMode = config.SupabaseModeLive
						metadata.SupabaseProjectID = project.ID
						metadata.SupabaseProjectRef = project.Ref

						jsonBytes, err := json.MarshalIndent(metadata, "", "  ")
						if err != nil {
							return "", fmt.Errorf("failed to marshal metadata: %w", err)
						}

						gh := api.NewGitHubClient(token, config.RemoteOrg)
						if err := gh.CreateOrUpdateFile(name, ".fpproto.json", "Upgrade to live Supabase", jsonBytes, metadataSHA); err != nil {
							return "", fmt.Errorf("failed to update metadata: %w", err)
						}

						return "mode: local -> live", nil
					},
				},
				{
					Title: "Updating local environment",
					Action: func() (string, error) {
						localDir := filepath.Join(config.PrototypesPath(), name)
						if _, err := os.Stat(localDir); os.IsNotExist(err) {
							return "not cloned locally, skipped", nil
						}

						envContent := fmt.Sprintf(
							"NEXT_PUBLIC_SUPABASE_URL=%s\nNEXT_PUBLIC_SUPABASE_ANON_KEY=%s\nSUPABASE_SERVICE_ROLE_KEY=%s\n",
							supabaseURL, anonKey, serviceRoleKey,
						)
						envPath := filepath.Join(localDir, ".env.local")
						if err := os.WriteFile(envPath, []byte(envContent), 0644); err != nil {
							return "", fmt.Errorf("failed to write .env.local: %w", err)
						}

						return ".env.local updated", nil
					},
				},
			})

			if err != nil {
				fmt.Printf("\n  %s Supabase project may have been partially created. Check the Supabase dashboard.\n\n", ui.WarningIcon)
				os.Exit(1)
			}

			summary := ui.RenderSummaryBox(
				name+" upgraded to live Supabase!",
				[][2]string{
					{"Supabase", supabaseURL},
					{"Project ref", project.Ref},
				},
				[]string{
					"Vercel will redeploy automatically with the new env vars.",
					"If cloned locally, restart your dev server to pick up the new .env.local.",
				},
			)
			fmt.Println(summary)
		},
	}
}
