package config

const (
	// ConfigDir is the directory under the user's home dir that stores CLI config.
	ConfigDir = ".fpproto"
	// ConfigFile is the filename for the local config.
	ConfigFile = "config.json"

	// RemoteOrg is the GitHub org that owns shared config and templates.
	RemoteOrg = "Flicent"
	// RemoteConfigRepo is the repo that stores the shared remote config.
	RemoteConfigRepo = ".fpproto-config"
	// RemoteConfigPath is the path inside RemoteConfigRepo to the config file.
	RemoteConfigPath = "config.json"

	// TemplateRepo is the name of the prototype template repository.
	TemplateRepo = "prototype-template"
	// PrototypesDir is the directory under the user's home dir for prototypes.
	PrototypesDir = "prototypes"
	// CLIRepo is the name of the CLI repository.
	CLIRepo = "fpproto"
)

// SupabaseMode constants.
const (
	SupabaseModeLocal = "local"
	SupabaseModeLive  = "live"
)

// Config represents the local CLI configuration stored on disk.
type Config struct {
	SupabaseAccessToken string `json:"supabase_access_token"`
	SupabaseOrgID       string `json:"supabase_org_id"`
	VercelToken         string `json:"vercel_token"`
	VercelTeamID        string `json:"vercel_team_id"`
	ConfigVersion       int    `json:"config_version"`
	UserEmail           string `json:"user_email"`
	SupabaseDeployHash  string `json:"supabase_deploy_hash,omitempty"`
}

// RemoteConfig represents the shared configuration fetched from the remote repo.
type RemoteConfig struct {
	SupabaseAccessToken string `json:"supabase_access_token"`
	SupabaseOrgID       string `json:"supabase_org_id"`
	VercelToken         string `json:"vercel_token"`
	VercelTeamID        string `json:"vercel_team_id"`
	ConfigVersion       int    `json:"config_version"`
	SupabaseDeployHash  string `json:"supabase_deploy_hash,omitempty"`
}

// PrototypeMetadata represents the .fpproto.json file stored in each prototype repo.
type PrototypeMetadata struct {
	PrototypeName      string `json:"prototype_name"`
	SupabaseMode       string `json:"supabase_mode"`
	SupabaseProjectID  string `json:"supabase_project_id,omitempty"`
	SupabaseProjectRef string `json:"supabase_project_ref,omitempty"`
	CreatedBy          string `json:"created_by"`
	CreatedAt          string `json:"created_at"`
}
