package cli

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"time"

	"github.com/fieldpulse-prototypes/fpproto/internal/api"
	"github.com/fieldpulse-prototypes/fpproto/internal/auth"
	"github.com/fieldpulse-prototypes/fpproto/internal/config"
	"github.com/fieldpulse-prototypes/fpproto/internal/ui"
	"github.com/spf13/cobra"
)

func NewCreateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "create <name>",
		Short: "Create a new prototype",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			name := args[0]

			validName := regexp.MustCompile(`^[a-z0-9]$|^[a-z0-9][a-z0-9-]*[a-z0-9]$`)
			if !validName.MatchString(name) {
				fmt.Println(ui.ErrorIcon + " Invalid name: must be lowercase alphanumeric with hyphens, cannot start or end with a hyphen")
				os.Exit(1)
			}

			fmt.Println(ui.HeaderStyle.Render("fpproto create"))
			fmt.Println()
			fmt.Printf("  Creating prototype %s...\n\n", ui.AccentStyle.Render(name))

			var email, token string
			var cfg *config.Config
			var project *api.Project
			var projectRef string
			var anonKey, serviceRoleKey, supabaseURL string
			var vercelProject *api.VercelProject

			err := ui.RunSteps([]ui.Step{
				{
					Title: "Authenticating",
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
					Title: "Checking config",
					Action: func() (string, error) {
						var err error
						cfg, err = config.Load()
						if err != nil {
							return "", fmt.Errorf("config not found, run fpproto setup first: %w", err)
						}

						// Auto-update check
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
									_ = config.Save(cfg)
								}
							}
						}

						return fmt.Sprintf("v%d", cfg.ConfigVersion), nil
					},
				},
				{
					Title: "Checking name availability",
					Action: func() (string, error) {
						gh := api.NewGitHubClient(token, config.RemoteOrg)
						repo, err := gh.GetRepo(name)
						if err != nil {
							return "", fmt.Errorf("failed to check repo: %w", err)
						}
						if repo != nil {
							return "", fmt.Errorf("repository %q already exists", name)
						}
						return "available", nil
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

						supabase := api.NewSupabaseClient(cfg.SupabaseAccessToken, cfg.SupabaseOrgID)
						var err error
						project, err = supabase.CreateProject(name, dbPass, "us-east-1")
						if err != nil {
							return "", fmt.Errorf("failed to create project: %w", err)
						}

						projectRef = project.Ref

						if err := supabase.WaitForProject(project.Ref, 120*time.Second); err != nil {
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

						supabase := api.NewSupabaseClient(cfg.SupabaseAccessToken, cfg.SupabaseOrgID)
						if err := supabase.RunSQL(projectRef, string(sql)); err != nil {
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

						supabase := api.NewSupabaseClient(cfg.SupabaseAccessToken, cfg.SupabaseOrgID)
						if err := supabase.RunSQL(projectRef, string(sql)); err != nil {
							return "", fmt.Errorf("seed data load failed: %w", err)
						}

						return "seed data loaded", nil
					},
				},
				{
					Title: "Creating GitHub repository",
					Action: func() (string, error) {
						gh := api.NewGitHubClient(token, config.RemoteOrg)
						if err := gh.CreateRepoFromTemplate(config.TemplateRepo, name, true); err != nil {
							return "", fmt.Errorf("failed to create repo: %w", err)
						}

						// Wait for GitHub to finish generating the repo from template
						time.Sleep(3 * time.Second)

						return "repository created", nil
					},
				},
				{
					Title: "Configuring prototype metadata",
					Action: func() (string, error) {
						metadata := config.PrototypeMetadata{
							PrototypeName:      name,
							SupabaseProjectID:  project.ID,
							SupabaseProjectRef: project.Ref,
							CreatedBy:          email,
							CreatedAt:          time.Now().UTC().Format(time.RFC3339),
						}

						jsonBytes, err := json.MarshalIndent(metadata, "", "  ")
						if err != nil {
							return "", fmt.Errorf("failed to marshal metadata: %w", err)
						}

						gh := api.NewGitHubClient(token, config.RemoteOrg)
						if err := gh.CreateOrUpdateFile(name, ".fpproto.json", "Add prototype metadata", jsonBytes, ""); err != nil {
							return "", fmt.Errorf("failed to commit metadata: %w", err)
						}

						return "metadata committed", nil
					},
				},
				{
					Title: "Fetching Supabase credentials",
					Action: func() (string, error) {
						supabase := api.NewSupabaseClient(cfg.SupabaseAccessToken, cfg.SupabaseOrgID)
						var err error
						anonKey, serviceRoleKey, err = supabase.GetAPIKeys(projectRef)
						if err != nil {
							return "", fmt.Errorf("failed to fetch API keys: %w", err)
						}
						supabaseURL = fmt.Sprintf("https://%s.supabase.co", projectRef)
						return "keys fetched", nil
					},
				},
				{
					Title: "Creating Vercel project",
					Action: func() (string, error) {
						vercel := api.NewVercelClient(cfg.VercelToken, cfg.VercelTeamID)
						var err error
						vercelProject, err = vercel.CreateProject(name, name, config.RemoteOrg)
						if err != nil {
							return "", fmt.Errorf("failed to create Vercel project: %w", err)
						}

						envVars := map[string]string{
							"NEXT_PUBLIC_SUPABASE_URL":      supabaseURL,
							"NEXT_PUBLIC_SUPABASE_ANON_KEY": anonKey,
							"SUPABASE_SERVICE_ROLE_KEY":     serviceRoleKey,
						}

						if err := vercel.SetEnvVars(vercelProject.ID, envVars); err != nil {
							return "", fmt.Errorf("failed to set env vars: %w", err)
						}

						return "project linked", nil
					},
				},
				{
					Title: "Cloning repository",
					Action: func() (string, error) {
						if err := config.EnsurePrototypesDir(); err != nil {
							return "", fmt.Errorf("failed to create prototypes directory: %w", err)
						}

						cloneDir := filepath.Join(config.PrototypesPath(), name)
						repoURL := fmt.Sprintf("https://github.com/%s/%s.git", config.RemoteOrg, name)

						cloneCmd := exec.Command("git", "clone", repoURL, cloneDir)
						cloneCmd.Stdout = nil
						cloneCmd.Stderr = nil
						if err := cloneCmd.Run(); err != nil {
							return "", fmt.Errorf("git clone failed: %w", err)
						}

						// Write .env.local
						envContent := fmt.Sprintf(
							"NEXT_PUBLIC_SUPABASE_URL=%s\nNEXT_PUBLIC_SUPABASE_ANON_KEY=%s\nSUPABASE_SERVICE_ROLE_KEY=%s\n",
							supabaseURL, anonKey, serviceRoleKey,
						)
						envPath := filepath.Join(cloneDir, ".env.local")
						if err := os.WriteFile(envPath, []byte(envContent), 0644); err != nil {
							return "", fmt.Errorf("failed to write .env.local: %w", err)
						}

						// Run npm install
						npmCmd := exec.Command("npm", "install")
						npmCmd.Dir = cloneDir
						npmCmd.Stdout = nil
						npmCmd.Stderr = nil
						if err := npmCmd.Run(); err != nil {
							return "", fmt.Errorf("npm install failed: %w", err)
						}

						return fmt.Sprintf("cloned to ~/prototypes/%s", name), nil
					},
				},
			})

			if err != nil {
				fmt.Printf("\n  %s If resources were partially created, run %s to clean up.\n\n",
					ui.WarningIcon, ui.AccentStyle.Render(fmt.Sprintf("fpproto destroy %s", name)))
				os.Exit(1)
			}

			vercelURL := ""
			if vercelProject != nil {
				vercelURL = vercelProject.URL()
			}

			summary := ui.RenderSummaryBox(
				name+" is ready!",
				[][2]string{
					{"Live URL", vercelURL},
					{"GitHub", fmt.Sprintf("https://github.com/%s/%s", config.RemoteOrg, name)},
					{"Local path", filepath.Join("~/prototypes", name)},
				},
				[]string{
					fmt.Sprintf("cd ~/prototypes/%s", name),
					"npm run dev",
				},
			)
			fmt.Println(summary)
		},
	}
}
