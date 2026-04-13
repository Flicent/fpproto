package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/fieldpulse-prototypes/fpproto/internal/api"
	"github.com/fieldpulse-prototypes/fpproto/internal/auth"
	"github.com/fieldpulse-prototypes/fpproto/internal/config"
	"github.com/fieldpulse-prototypes/fpproto/internal/ui"
	"github.com/spf13/cobra"
)

func NewCloneCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "clone <name>",
		Short: "Clone an existing prototype locally",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			name := args[0]

			fmt.Println(ui.HeaderStyle.Render("fpproto clone"))
			fmt.Println()
			fmt.Printf("  Cloning prototype %s...\n\n", ui.AccentStyle.Render(name))

			var email, token string
			var cfg *config.Config
			var metadata config.PrototypeMetadata
			var anonKey, serviceRoleKey, supabaseURL string
			var vercelURL string

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
						return fmt.Sprintf("v%d", cfg.ConfigVersion), nil
					},
				},
				{
					Title: "Verifying prototype exists",
					Action: func() (string, error) {
						gh := api.NewGitHubClient(token, config.RemoteOrg)
						repo, err := gh.GetRepo(name)
						if err != nil {
							return "", fmt.Errorf("failed to check repo: %w", err)
						}
						if repo == nil {
							return "", fmt.Errorf("prototype %q not found", name)
						}
						if repo.Archived {
							return "", fmt.Errorf("prototype %q is archived and read-only", name)
						}
						return "found", nil
					},
				},
				{
					Title: "Checking local directory",
					Action: func() (string, error) {
						localPath := filepath.Join(config.PrototypesPath(), name)
						if _, err := os.Stat(localPath); err == nil {
							return "", fmt.Errorf("directory already exists: %s — remove it first or use a different name", localPath)
						}
						return "ready", nil
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

						return fmt.Sprintf("cloned to ~/prototypes/%s", name), nil
					},
				},
				{
					Title: "Reading prototype metadata",
					Action: func() (string, error) {
						metadataPath := filepath.Join(config.PrototypesPath(), name, ".fpproto.json")
						data, err := os.ReadFile(metadataPath)
						if err != nil {
							return "", fmt.Errorf("failed to read .fpproto.json: %w", err)
						}
						if err := json.Unmarshal(data, &metadata); err != nil {
							return "", fmt.Errorf("failed to parse .fpproto.json: %w", err)
						}
						return metadata.SupabaseProjectRef, nil
					},
				},
				{
					Title: "Fetching Supabase credentials",
					Action: func() (string, error) {
						supabase := api.NewSupabaseClient(cfg.SupabaseAccessToken, cfg.SupabaseOrgID)
						var err error
						anonKey, serviceRoleKey, err = supabase.GetAPIKeys(metadata.SupabaseProjectRef)
						if err != nil {
							return "", fmt.Errorf("failed to fetch API keys: %w", err)
						}
						supabaseURL = fmt.Sprintf("https://%s.supabase.co", metadata.SupabaseProjectRef)
						return "credentials fetched", nil
					},
				},
				{
					Title: "Writing .env.local",
					Action: func() (string, error) {
						envContent := fmt.Sprintf(
							"NEXT_PUBLIC_SUPABASE_URL=%s\nNEXT_PUBLIC_SUPABASE_ANON_KEY=%s\nSUPABASE_SERVICE_ROLE_KEY=%s\n",
							supabaseURL, anonKey, serviceRoleKey,
						)
						envPath := filepath.Join(config.PrototypesPath(), name, ".env.local")
						if err := os.WriteFile(envPath, []byte(envContent), 0644); err != nil {
							return "", fmt.Errorf("failed to write .env.local: %w", err)
						}
						return "environment configured", nil
					},
				},
				{
					Title: "Installing dependencies",
					Action: func() (string, error) {
						npmCmd := exec.Command("npm", "install")
						npmCmd.Dir = filepath.Join(config.PrototypesPath(), name)
						npmCmd.Stdout = nil
						npmCmd.Stderr = nil
						if err := npmCmd.Run(); err != nil {
							return "", fmt.Errorf("npm install failed: %w", err)
						}
						return "npm install complete", nil
					},
				},
				{
					Title: "Looking up deployment",
					Action: func() (string, error) {
						vercel := api.NewVercelClient(cfg.VercelToken, cfg.VercelTeamID)
						vp, err := vercel.GetProject(name)
						if err != nil {
							return "", fmt.Errorf("failed to look up Vercel project: %w", err)
						}
						if vp != nil {
							vercelURL = vp.URL()
							return vercelURL, nil
						}
						vercelURL = "not deployed"
						return "deployment found", nil
					},
				},
			})

			if err != nil {
				os.Exit(1)
			}

			summary := ui.RenderSummaryBox(
				name+" cloned!",
				[][2]string{
					{"Live URL", vercelURL},
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
