# fpproto

A Go CLI that lets the product team create, clone, and archive prototyping environments with a single command. It orchestrates Supabase, GitHub, and Vercel APIs so product never touches a dashboard or manages credentials.

## Install

```bash
curl -fsSL https://raw.githubusercontent.com/Flicent/fpproto/main/install.sh | sudo bash
```

Or build from source:

```bash
go build -o fpproto ./cmd/fpproto
```

## Uninstall

```bash
curl -fsSL https://raw.githubusercontent.com/Flicent/fpproto/main/uninstall.sh | sudo bash
```

## Prerequisites

- [GitHub CLI](https://cli.github.com/) installed and authenticated (`brew install gh && gh auth login`)
- Access to the `fieldpulse-prototypes` GitHub org
- [Docker Desktop](https://docs.docker.com/get-docker/) installed and running (for local Supabase)
- [Supabase CLI](https://supabase.com/docs/guides/cli) installed (`brew install supabase/tap/supabase`)

## Setup

Run once per machine to pull org credentials:

```bash
fpproto setup
```

## Supabase Modes

Prototypes support two Supabase modes:

- **Local (default)** — Supabase runs locally via Docker. Free, fast, and sufficient for most prototyping. No cloud resources are provisioned.
- **Live** — A real Supabase cloud project is created. Requires an admin deploy password. Use only when the prototype needs a shared, persistent database (roughly 5% of the time).

You can start local and upgrade to live later with `fpproto supabase <name>`.

## Commands

### `fpproto create <name>`

Creates a new prototype environment — GitHub repo (from template), Vercel deployment, and local clone. By default, Supabase runs locally via Docker.

```
  fpproto create billing-redesign

  ✓ Authenticating (sarah@fieldpulse.com)
  ✓ Checking Docker (Docker running, Supabase CLI found)
  ✓ Creating GitHub repository (repository created)
  ✓ Cloning repository (cloned to ~/prototypes/billing-redesign)
  ✓ Starting local Supabase (running at http://127.0.0.1:54321)
  ✓ Creating Vercel project (project linked)

  ┌─────────────────────────────────────────────────────┐
  │                                                     │
  │  billing-redesign is ready! (local Supabase)        │
  │                                                     │
  │  Live URL    https://billing-redesign.vercel.app    │
  │  GitHub      https://github.com/Flicent/...         │
  │  Local       ~/prototypes/billing-redesign          │
  │  Supabase    local (Docker)                         │
  │                                                     │
  │  Next steps:                                        │
  │    cd ~/prototypes/billing-redesign                 │
  │    npm run dev                                      │
  │                                                     │
  └─────────────────────────────────────────────────────┘
```

To create with a live Supabase project instead (requires admin password):

```bash
fpproto create billing-redesign --live
```

Name must be lowercase alphanumeric with hyphens (`^[a-z0-9][a-z0-9-]*[a-z0-9]$`).

### `fpproto supabase <name>`

Upgrades an existing local-mode prototype to use a live Supabase cloud project. Requires the admin deploy password.

```
  fpproto supabase billing-redesign

  ✓ Reading prototype metadata (local mode confirmed)
  ✓ Verifying deploy authorization (authorized)
  ✓ Creating Supabase project (region: us-east-1)
  ✓ Running migrations (schema created)
  ✓ Loading seed data (seed data loaded)
  ✓ Updating Vercel environment (env vars updated)
  ✓ Updating prototype metadata (mode: local -> live)
  ✓ Updating local environment (.env.local updated)
```

This provisions the cloud project, runs migrations, pushes credentials to Vercel, and updates both the repo metadata and your local `.env.local`.

### `fpproto clone <name>`

Pulls an existing prototype onto the current machine. Automatically detects the Supabase mode — local-mode prototypes start Docker containers, live-mode prototypes fetch cloud credentials.

```
  fpproto clone scheduling-v2

  ✓ Cloning repository (cloned to ~/prototypes/scheduling-v2)
  ✓ Setting up Supabase (local instance at http://127.0.0.1:54321)
  ✓ .env.local written
  ✓ npm install complete
```

### `fpproto destroy <name>`

Archives a prototype. Makes the repo read-only and deletes infrastructure. For live-mode prototypes, the Supabase cloud project is deleted. For local-mode prototypes, only the Vercel deployment is removed (stop local containers with `supabase stop`). Requires typing the name to confirm.

```
  fpproto destroy billing-redesign

  ⚠  This will:
    • Archive the GitHub repo (read-only, code preserved)
    • Delete the Supabase project (if live mode — data gone permanently)
    • Delete the Vercel deployment

  Type "billing-redesign" to confirm:
```

### `fpproto list`

Lists all active prototypes in the org with their Supabase mode.

```
  Active Prototypes (3)

  NAME                  CREATED BY              CREATED         SUPABASE    URL
  billing-redesign      sarah@fieldpulse.com    3 days ago      local       https://billing-redesign.vercel.app
  scheduling-v2         owen@fieldpulse.com     1 week ago      live        https://scheduling-v2.vercel.app
  new-onboarding        mike@fieldpulse.com     2 weeks ago     local       https://new-onboarding.vercel.app
```

### `fpproto update`

Self-updates the CLI binary from GitHub Releases.

A version nudge is shown automatically after any command when a newer version is available.

## Admin: Deploy Password

Live Supabase deployments are gated by an admin password to control costs. The bcrypt hash of this password is stored in the remote org config (`Flicent/.fpproto-config/config.json`) under the `supabase_deploy_hash` field.

To generate a hash:

```bash
htpasswd -nbBC 10 "" "your-password" | cut -d: -f2
```

Add it to the remote config:

```json
{
  "supabase_access_token": "...",
  "supabase_org_id": "...",
  "vercel_token": "...",
  "vercel_team_id": "...",
  "config_version": 2,
  "supabase_deploy_hash": "$2y$10$..."
}
```

All team members will pick up the hash on their next command via the config auto-update.

## Config

Local config is stored at `~/.fpproto/config.json` (created by `fpproto setup`). It contains org API tokens pulled from a private config repo — never commit or share this file.

Prototypes are cloned to `~/prototypes/<name>`.

## Build & Release

```bash
# Dev build
go build -o fpproto ./cmd/fpproto

# Production build with version
go build -ldflags "-X main.version=v1.0.0" -o fpproto ./cmd/fpproto
```

Releases are automated via GitHub Actions — pushing to main builds darwin/arm64 and darwin/amd64 binaries and creates a GitHub Release. Releases are currently paused and can be triggered manually from the Actions tab.

## Project Structure

```
fpproto/
├── cmd/fpproto/main.go          # Entry point, root Cobra command
├── internal/
│   ├── cli/                     # Command implementations (setup, create, clone, destroy, list, update, supabase)
│   ├── api/                     # API clients (Supabase, GitHub, Vercel)
│   ├── auth/                    # GitHub CLI auth helpers
│   ├── config/                  # Config types, load/save, constants
│   ├── docker/                  # Docker availability checks
│   ├── supabase/                # Local Supabase CLI wrapper (init, start, status, stop)
│   └── ui/                      # Terminal styling, spinners, tables, boxes, confirmation prompts
├── install.sh                   # curl install script
├── .github/workflows/release.yml
├── go.mod
└── go.sum
```
