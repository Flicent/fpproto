# fpproto CLI - Implementation Plan

## Overview

`fpproto` is a Go CLI tool that lets the product team create, clone, and archive prototyping environments with a single command. It orchestrates Supabase, GitHub, and Vercel APIs so product never touches a dashboard or manages credentials.

The CLI should feel polished and professional. Use colors, spinners, and structured output throughout. This is a tool for non-engineers who will judge it by how it looks and feels.

-----

## Tech Stack

- **Language:** Go
- **CLI framework:** [Cobra](https://github.com/spf13/cobra)
- **Terminal UI:** [Charm’s Lip Gloss](https://github.com/charmbracelet/lipgloss) for styling, [Bubble Tea](https://github.com/charmbracelet/bubbletea) for interactive elements, [Huh](https://github.com/charmbracelet/huh) for forms/prompts
- **Spinners:** [Charm’s spinner](https://github.com/charmbracelet/bubbles/tree/master/spinner) from Bubbles
- **Progress:** [Charm’s progress](https://github.com/charmbracelet/bubbles/tree/master/progress) from Bubbles
- **Auth:** GitHub CLI (`gh`) as the sole identity provider
- **Config:** JSON file cached at `~/.fpproto/config.json`
- **Distribution:** Public GitHub Releases with a curl install script

-----

## Visual Design

### Color Palette

|Element|Color              |Usage                              |
|-------|-------------------|-----------------------------------|
|Primary|Cyan (`#00D4FF`)   |Headers, command names, URLs       |
|Success|Green (`#00FF88`)  |Completion messages, checkmarks    |
|Warning|Yellow (`#FFD700`) |Prompts, version nudges            |
|Error  |Red (`#FF4444`)    |Failures, destructive confirmations|
|Muted  |Gray (`#888888`)   |Secondary info, timestamps, hints  |
|Accent |Magenta (`#FF44FF`)|Prototype names, emphasis          |

### Conventions

- Every command starts with a styled header showing the command name
- Multi-step operations show a spinner with a description for each step
- Completed steps show a green checkmark (`✓`) with the result
- Failed steps show a red cross (`✗`) with the error
- URLs are always cyan and underlined
- Prototype names are always magenta and bold
- Final output is boxed with a border using Lip Gloss

### Example Output: `fpproto create`

```
  fpproto create

  Creating prototype billing-redesign...

  ✓ Authenticated as sarah@fieldpulse.com
  ✓ Config up to date (v3)
  ⠋ Creating Supabase project...
  ✓ Supabase project created (region: us-east-1)
  ⠋ Running migrations...
  ✓ Schema created (6 tables)
  ✓ Seed data loaded (94 records)
  ⠋ Creating GitHub repository...
  ✓ Repository created
  ⠋ Deploying to Vercel...
  ✓ Vercel project linked and deploying

  ┌─────────────────────────────────────────────────┐
  │                                                 │
  │  billing-redesign is ready!                     │
  │                                                 │
  │  Live URL    https://billing-redesign.vercel.app│
  │  GitHub      https://github.com/fp-proto/...    │
  │  Local       ~/prototypes/billing-redesign      │
  │                                                 │
  │  Next steps:                                    │
  │    cd ~/prototypes/billing-redesign             │
  │    npm run dev                                  │
  │                                                 │
  └─────────────────────────────────────────────────┘
```

### Example Output: `fpproto destroy`

```
  fpproto destroy

  ⚠  You are about to archive billing-redesign

  This will:
    • Archive the GitHub repo (read-only, code preserved)
    • Delete the Supabase project (data gone permanently)
    • Delete the Vercel deployment

  Type "billing-redesign" to confirm:  █

  ✓ README updated with archive banner
  ✓ GitHub repo archived
  ✓ Supabase project deleted
  ✓ Vercel project deleted

  billing-redesign has been archived.
```

### Example Output: `fpproto list`

```
  fpproto list

  Active Prototypes (3)

  NAME                  CREATED BY              CREATED         URL
  billing-redesign      sarah@fieldpulse.com    3 days ago      https://billing-redesign.vercel.app
  scheduling-v2         owen@fieldpulse.com     1 week ago      https://scheduling-v2.vercel.app
  new-onboarding        mike@fieldpulse.com     2 weeks ago     https://new-onboarding.vercel.app
```

### Example Output: `fpproto clone`

```
  fpproto clone

  Cloning prototype scheduling-v2...

  ✓ Authenticated as owen@fieldpulse.com
  ✓ Repository cloned to ~/prototypes/scheduling-v2
  ✓ Supabase credentials fetched
  ✓ .env.local written
  ✓ npm install complete

  ┌─────────────────────────────────────────────────┐
  │                                                 │
  │  scheduling-v2 is ready!                        │
  │                                                 │
  │  Live URL    https://scheduling-v2.vercel.app   │
  │  Local       ~/prototypes/scheduling-v2         │
  │                                                 │
  │  Next steps:                                    │
  │    cd ~/prototypes/scheduling-v2                │
  │    npm run dev                                  │
  │                                                 │
  └─────────────────────────────────────────────────┘
```

### Example Output: `fpproto setup`

```
  fpproto setup

  ✓ GitHub CLI found
  ✓ Authenticated as sarah@fieldpulse.com
  ⠋ Pulling org config...
  ✓ Config cached (v3)
  ✓ Supabase API connected
  ✓ Vercel API connected

  You're all set! Run fpproto create <name> to start a prototype.
```

### Example Output: `fpproto update`

```
  fpproto update

  ⠋ Checking for updates...
  ✓ New version available: v1.3.0 (current: v1.2.1)
  ⠋ Downloading...
  ✓ Updated to v1.3.0

  Changelog:
    • Added fpproto clone command
    • Fixed Supabase project region selection
```

### Version Nudge (shown on any command when outdated)

```
  ┌──────────────────────────────────────────────┐
  │  Update available: v1.2.1 → v1.3.0          │
  │  Run fpproto update to upgrade               │
  └──────────────────────────────────────────────┘
```

-----

## Commands

### `fpproto setup`

First-time machine configuration. Validates prerequisites and pulls org secrets.

**Prerequisites:** `gh` CLI installed and authenticated.

**Flow:**

1. Check `gh` is installed. If not, print:
   
   ```
   ✗ GitHub CLI not found.
     Install it: brew install gh
     Then run: gh auth login
   ```
   
   Exit with code 1.
1. Check `gh auth status`. If not logged in, print:
   
   ```
   ✗ Not logged into GitHub.
     Run: gh auth login
   ```
   
   Exit with code 1.
1. Get the authenticated user’s email from `gh api user`.
1. Use `gh` to fetch `config.json` from `fieldpulse-prototypes/.fpproto-config` via GitHub API (`GET /repos/{owner}/{repo}/contents/{path}`). If 404 or 403, print:
   
   ```
   ✗ You don't have access to the fpproto config.
     Ask engineering to add you to the fieldpulse-prototypes org.
   ```
   
   Exit with code 1.
1. Decode the base64 response content and parse JSON.
1. Write to `~/.fpproto/config.json`.
1. Validate connections:
- Hit Supabase Management API (`GET /v1/projects`) with the org token. If fails, print error.
- Hit Vercel API (`GET /v2/teams/{teamId}`) with the team token. If fails, print error.
1. Print success message.

**Config file format (`~/.fpproto/config.json`):**

```json
{
  "supabase_access_token": "sbp_...",
  "supabase_org_id": "org-...",
  "vercel_token": "...",
  "vercel_team_id": "...",
  "config_version": 3,
  "user_email": "sarah@fieldpulse.com"
}
```

-----

### `fpproto create <name>`

Creates a new prototype environment from scratch.

**Arguments:**

- `name` (required): Prototype name. Must be lowercase alphanumeric with hyphens. Used as the repo name, Supabase project name, and Vercel project name.

**Validation:**

- Name must match `^[a-z0-9][a-z0-9-]*[a-z0-9]$` (no leading/trailing hyphens, no uppercase, no spaces)
- Name must not already exist as a repo in the org (check via GitHub API)
- Config must exist and be current

**Flow:**

1. **Auth check:** Verify `gh` auth and load config. Auto-update config if stale.
1. **Create Supabase project:**
- `POST /v1/projects` with body:
  
  ```json
  {
    "name": "<name>",
    "organization_id": "<org_id>",
    "db_pass": "<generated_password>",
    "region": "us-east-1",
    "plan": "pro"
  }
  ```
- Poll `GET /v1/projects/{id}` until status is `ACTIVE_HEALTHY` (timeout after 120 seconds)
- Store the project `id` and `ref`
1. **Run migrations:**
- Fetch the migration SQL from the template repo (`supabase/migrations/001_seed_schema.sql`)
- Execute against the Supabase database using the Management API SQL endpoint or `pg` connection string
- Alternative: use the Supabase CLI programmatically if available
1. **Run seed data:**
- Fetch `supabase/seed.sql` from the template repo
- Execute against the database
1. **Create GitHub repo:**
- `POST /repos/{org}/generate` using the template repo
- Repo name: `<name>`
- Private repo within the org
1. **Commit `.fpproto.json`:**
- Create file via GitHub Contents API (`PUT /repos/{org}/{name}/contents/.fpproto.json`)
- Contents:
  
  ```json
  {
    "prototype_name": "<name>",
    "supabase_project_id": "<project_id>",
    "supabase_project_ref": "<project_ref>",
    "created_by": "<user_email>",
    "created_at": "<ISO 8601 timestamp>"
  }
  ```
1. **Fetch Supabase credentials:**
- `GET /v1/projects/{ref}/api-keys` to get `anon` and `service_role` keys
- Construct the Supabase URL: `https://<ref>.supabase.co`
1. **Create Vercel project:**
- `POST /v1/projects` with body linking to the GitHub repo
- Set environment variables:
  - `NEXT_PUBLIC_SUPABASE_URL`
  - `NEXT_PUBLIC_SUPABASE_ANON_KEY`
  - `SUPABASE_SERVICE_ROLE_KEY`
1. **Trigger deploy:**
- Vercel auto-deploys from the linked repo, but trigger an initial deployment via `POST /v1/deployments` if needed
1. **Clone locally:**
- `git clone` to `~/prototypes/<name>`
- Write `.env.local` with the Supabase credentials (this file is in `.gitignore`)
- Run `npm install`
1. **Print summary box** with live URL, GitHub URL, local path, and next steps.

**Error handling:**

- If any step fails, print which step failed and what was already created
- Do not auto-rollback (partial state is better than silent cleanup that might also fail)
- Print a hint: “Run `fpproto destroy <name>` to clean up, or fix the issue and try again.”

-----

### `fpproto clone <name>`

Pulls an existing prototype onto the current machine. Used when switching laptops or when a second person joins a prototype.

**Arguments:**

- `name` (required): Name of an existing prototype repo in the org.

**Validation:**

- Repo must exist in the org and not be archived
- `~/prototypes/<name>` must not already exist locally (if it does, print hint to `cd` into it or delete it first)

**Flow:**

1. **Auth check:** Verify `gh` auth and load config. Auto-update config if stale.
1. **Verify repo exists:**
- `GET /repos/{org}/{name}` via GitHub API
- If 404: “Prototype not found. Run `fpproto list` to see active prototypes.”
- If archived: “This prototype has been archived and is read-only.”
1. **Clone repo:**
- `git clone` to `~/prototypes/<name>`
1. **Read `.fpproto.json`:**
- Parse the file from the cloned repo to get `supabase_project_id` and `supabase_project_ref`
1. **Fetch Supabase credentials:**
- `GET /v1/projects/{ref}/api-keys` using the org access token
- Construct the Supabase URL from the ref
1. **Write `.env.local`:**
- Write Supabase credentials to `.env.local` in the project root (gitignored)
1. **Install dependencies:**
- Run `npm install` in the project directory
1. **Get Vercel URL:**
- Look up the Vercel project by name or by linked GitHub repo to get the deployment URL
1. **Print summary box** with live URL, local path, and next steps.

-----

### `fpproto destroy <name>`

Archives a prototype. Makes the repo read-only, deletes infrastructure.

**Arguments:**

- `name` (required): Name of the prototype to archive.

**Validation:**

- Repo must exist and not already be archived
- Supabase project must exist (or be already deleted, which is fine)

**Flow:**

1. **Auth check:** Verify `gh` auth and load config.
1. **Confirm:** Show a warning box listing what will happen. Require the user to type the prototype name to confirm (not just “y”). Use Huh’s input form with validation that the typed name matches exactly.
1. **Update README:**
- Fetch current README via GitHub Contents API
- Prepend the archive banner with date and user email
- Commit via Contents API with message: “Archive: prototype archived by <email>”
1. **Archive GitHub repo:**
- `PATCH /repos/{org}/{name}` with `{ "archived": true }`
1. **Delete Supabase project:**
- Read `.fpproto.json` from the repo to get the project ref
- `DELETE /v1/projects/{ref}`
- If already deleted (404), skip with a note
1. **Delete Vercel project:**
- Look up project by name
- `DELETE /v1/projects/{id}`
- If already deleted (404), skip with a note
1. **Print confirmation.**

-----

### `fpproto list`

Lists all active prototypes in the org.

**Flow:**

1. **Auth check:** Verify `gh` auth and load config.
1. **Fetch repos:**
- `GET /orgs/{org}/repos` with `type=all` and filter out archived repos
- Also filter out non-prototype repos (the config repo, the template repo, the CLI repo) by checking for `.fpproto.json` or by name convention
1. **For each repo:**
- Read `.fpproto.json` to get `created_by` and `created_at`
- Look up the Vercel deployment URL
1. **Render table** with columns: Name, Created By, Created, URL
- Use Lip Gloss for styled table output
- Sort by creation date (most recent first)
- Show relative timestamps (“3 days ago”, “1 week ago”)
1. If no active prototypes, print: “No active prototypes. Run `fpproto create <name>` to start one.”

-----

### `fpproto update`

Self-updates the CLI binary.

**Flow:**

1. **Check latest release:**
- `GET /repos/{org}/fpproto/releases/latest` (public repo, no auth needed)
- Compare version tag with current binary version (compiled in at build time via `-ldflags`)
1. **If up to date:** Print “Already on the latest version (v1.3.0).”
1. **If update available:**
- Determine platform and architecture (`darwin-arm64`, `darwin-amd64`)
- Download the binary from the release assets
- Replace the current binary (write to a temp file, then `os.Rename`)
- Print the new version and changelog (from release body)

**Version check on every command:**

- On every command (except `update` itself), spawn a background goroutine that checks the latest release
- If a new version is found, print the version nudge box after the command output
- The check must be non-blocking and fail silently (no errors if GitHub is unreachable)
- Cache the last check timestamp in `~/.fpproto/last_update_check` to avoid checking more than once per hour

-----

## Config Management

### Local Config (`~/.fpproto/config.json`)

```json
{
  "supabase_access_token": "sbp_...",
  "supabase_org_id": "org-...",
  "vercel_token": "...",
  "vercel_team_id": "...",
  "config_version": 3,
  "user_email": "sarah@fieldpulse.com"
}
```

### Remote Config (`fieldpulse-prototypes/.fpproto-config/config.json`)

```json
{
  "supabase_access_token": "sbp_...",
  "supabase_org_id": "org-...",
  "vercel_token": "...",
  "vercel_team_id": "...",
  "config_version": 3
}
```

### Auto-Update Logic

Every command that needs config runs this check:

1. Load local config from `~/.fpproto/config.json`
1. Fetch remote config version from GitHub API (just the `config_version` field, not the full file)
1. If remote version > local version:
- Fetch full remote config
- Overwrite local config
- Print: “Config updated (v2 -> v3).”
1. If fetch fails (network issue), proceed with local config silently

-----

## API Reference

### Supabase Management API

Base URL: `https://api.supabase.com`

Auth header: `Authorization: Bearer <supabase_access_token>`

|Operation     |Method|Endpoint                           |
|--------------|------|-----------------------------------|
|List projects |GET   |`/v1/projects`                     |
|Create project|POST  |`/v1/projects`                     |
|Get project   |GET   |`/v1/projects/{ref}`               |
|Delete project|DELETE|`/v1/projects/{ref}`               |
|Get API keys  |GET   |`/v1/projects/{ref}/api-keys`      |
|Run SQL       |POST  |`/v1/projects/{ref}/database/query`|

### GitHub REST API

Base URL: `https://api.github.com`

Auth: via `gh auth token` passed as `Authorization: Bearer <token>`

|Operation                |Method|Endpoint                                          |
|-------------------------|------|--------------------------------------------------|
|Get authenticated user   |GET   |`/user`                                           |
|Create repo from template|POST  |`/repos/{template_owner}/{template_repo}/generate`|
|Get repo                 |GET   |`/repos/{owner}/{repo}`                           |
|List org repos           |GET   |`/orgs/{org}/repos`                               |
|Archive repo             |PATCH |`/repos/{owner}/{repo}` with `{"archived": true}` |
|Get file contents        |GET   |`/repos/{owner}/{repo}/contents/{path}`           |
|Create/update file       |PUT   |`/repos/{owner}/{repo}/contents/{path}`           |
|Get latest release       |GET   |`/repos/{owner}/{repo}/releases/latest`           |

### Vercel REST API

Base URL: `https://api.vercel.com`

Auth header: `Authorization: Bearer <vercel_token>`

|Operation        |Method|Endpoint               |
|-----------------|------|-----------------------|
|Get team         |GET   |`/v2/teams/{teamId}`   |
|Create project   |POST  |`/v1/projects`         |
|Delete project   |DELETE|`/v1/projects/{id}`    |
|Get project      |GET   |`/v1/projects/{id}`    |
|Set env vars     |POST  |`/v1/projects/{id}/env`|
|Create deployment|POST  |`/v1/deployments`      |

-----

## Build and Release

### Build

```bash
# Local dev build
go build -o fpproto ./cmd/fpproto

# Production build with version baked in
go build -ldflags "-X main.version=v1.0.0" -o fpproto ./cmd/fpproto
```

### Cross-Compile Targets

|Platform           |GOOS  |GOARCH|Binary Name           |
|-------------------|------|------|----------------------|
|macOS Apple Silicon|darwin|arm64 |`fpproto-darwin-arm64`|
|macOS Intel        |darwin|amd64 |`fpproto-darwin-amd64`|

Linux targets can be added later if needed, but product is all on macOS.

### GitHub Actions Release Workflow

Triggered on tag push matching `v*`:

1. Check out code
1. Set up Go
1. Cross-compile for both darwin targets
1. Create GitHub Release with the tag
1. Upload both binaries as release assets
1. Include changelog from tag annotation or `CHANGELOG.md`

### Install Script (`install.sh`)

Hosted at `https://raw.githubusercontent.com/fieldpulse-prototypes/fpproto/main/install.sh`

```bash
#!/bin/bash
set -euo pipefail

REPO="fieldpulse-prototypes/fpproto"
INSTALL_DIR="/usr/local/bin"

# Detect architecture
ARCH=$(uname -m)
case "$ARCH" in
  arm64|aarch64) BINARY="fpproto-darwin-arm64" ;;
  x86_64)        BINARY="fpproto-darwin-amd64" ;;
  *)             echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

# Get latest release URL
LATEST=$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" | grep "browser_download_url.*$BINARY" | cut -d '"' -f 4)

if [ -z "$LATEST" ]; then
  echo "Failed to find latest release."
  exit 1
fi

echo "Downloading fpproto..."
curl -fsSL "$LATEST" -o "$INSTALL_DIR/fpproto"
chmod +x "$INSTALL_DIR/fpproto"

echo "fpproto installed to $INSTALL_DIR/fpproto"
fpproto --version
```

-----

## Project Structure

```
fpproto/
├── cmd/
│   └── fpproto/
│       └── main.go              # Entry point, Cobra root command
├── internal/
│   ├── cli/
│   │   ├── setup.go             # fpproto setup
│   │   ├── create.go            # fpproto create
│   │   ├── clone.go             # fpproto clone
│   │   ├── destroy.go           # fpproto destroy
│   │   ├── list.go              # fpproto list
│   │   └── update.go            # fpproto update
│   ├── api/
│   │   ├── supabase.go          # Supabase Management API client
│   │   ├── github.go            # GitHub REST API client
│   │   └── vercel.go            # Vercel REST API client
│   ├── auth/
│   │   └── gh.go                # GitHub CLI auth helpers
│   ├── config/
│   │   ├── config.go            # Config load/save/update logic
│   │   └── types.go             # Config struct definitions
│   └── ui/
│       ├── styles.go            # Lip Gloss style definitions (colors, borders)
│       ├── spinner.go           # Spinner wrapper for multi-step operations
│       ├── table.go             # Styled table rendering
│       ├── box.go               # Summary box rendering
│       └── confirm.go           # Destructive action confirmation prompt
├── install.sh                   # Public install script
├── go.mod
├── go.sum
├── .goreleaser.yaml             # Optional: GoReleaser config as alternative to manual Actions
└── README.md
```

-----

## Error Handling

### Principles

- Never silently fail. Always tell the user what went wrong and what to do about it.
- On multi-step operations, show which steps succeeded before the failure.
- Do not auto-rollback partial state. Print what was created so the user (or `fpproto destroy`) can clean up.
- API errors should show the HTTP status and a human-readable message, not raw JSON.

### Common Errors

|Error                     |Message                                                                                                      |
|--------------------------|-------------------------------------------------------------------------------------------------------------|
|`gh` not installed        |`✗ GitHub CLI not found. Install it: brew install gh`                                                        |
|Not logged into `gh`      |`✗ Not logged into GitHub. Run: gh auth login`                                                               |
|No org access             |`✗ You don't have access to the fpproto config. Ask engineering to add you to the fieldpulse-prototypes org.`|
|Prototype name taken      |`✗ A prototype named "billing-redesign" already exists. Run fpproto list to see active prototypes.`          |
|Prototype not found       |`✗ No prototype named "billing-redesign" found. Run fpproto list to see active prototypes.`                  |
|Prototype already archived|`✗ "billing-redesign" is already archived.`                                                                  |
|Supabase API error        |`✗ Supabase error (403): Invalid access token. Run fpproto setup to refresh your config.`                    |
|Vercel API error          |`✗ Vercel error (429): Rate limited. Wait a moment and try again.`                                           |
|Network failure           |`✗ Could not reach api.supabase.com. Check your internet connection.`                                        |
|Local dir exists          |`✗ ~/prototypes/billing-redesign already exists. Delete it or choose a different name.`                      |
|npm install fails         |`⚠ npm install failed. You may need to run it manually: cd ~/prototypes/<name> && npm install`               |
