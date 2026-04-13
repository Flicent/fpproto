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

## Setup

Run once per machine to pull org credentials:

```bash
fpproto setup
```

## Commands

### `fpproto create <name>`

Creates a new prototype environment from scratch — Supabase project, GitHub repo (from template), Vercel deployment, and local clone.

```
  fpproto create billing-redesign

  ✓ Authenticating (sarah@fieldpulse.com)
  ✓ Creating Supabase project (region: us-east-1)
  ✓ Running migrations (schema created)
  ✓ Creating GitHub repository (repository created)
  ✓ Creating Vercel project (project linked)
  ✓ Cloning repository (cloned to ~/prototypes/billing-redesign)

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

Name must be lowercase alphanumeric with hyphens (`^[a-z0-9][a-z0-9-]*[a-z0-9]$`).

### `fpproto clone <name>`

Pulls an existing prototype onto the current machine. Used when switching laptops or when a second person joins.

```
  fpproto clone scheduling-v2

  ✓ Repository cloned to ~/prototypes/scheduling-v2
  ✓ Supabase credentials fetched
  ✓ .env.local written
  ✓ npm install complete
```

### `fpproto destroy <name>`

Archives a prototype. Makes the repo read-only and deletes Supabase + Vercel infrastructure. Requires typing the name to confirm.

```
  fpproto destroy billing-redesign

  ⚠  This will:
    • Archive the GitHub repo (read-only, code preserved)
    • Delete the Supabase project (data gone permanently)
    • Delete the Vercel deployment

  Type "billing-redesign" to confirm:
```

### `fpproto list`

Lists all active prototypes in the org.

```
  Active Prototypes (3)

  NAME                  CREATED BY              CREATED         URL
  billing-redesign      sarah@fieldpulse.com    3 days ago      https://billing-redesign.vercel.app
  scheduling-v2         owen@fieldpulse.com     1 week ago      https://scheduling-v2.vercel.app
  new-onboarding        mike@fieldpulse.com     2 weeks ago     https://new-onboarding.vercel.app
```

### `fpproto update`

Self-updates the CLI binary from GitHub Releases.

A version nudge is shown automatically after any command when a newer version is available.

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

Releases are automated via GitHub Actions — push a tag matching `v*` to build darwin/arm64 and darwin/amd64 binaries and create a GitHub Release.

## Project Structure

```
fpproto/
├── cmd/fpproto/main.go          # Entry point, root Cobra command
├── internal/
│   ├── cli/                     # Command implementations (setup, create, clone, destroy, list, update)
│   ├── api/                     # API clients (Supabase, GitHub, Vercel)
│   ├── auth/                    # GitHub CLI auth helpers
│   ├── config/                  # Config types, load/save, constants
│   └── ui/                      # Terminal styling, spinners, tables, boxes, confirmation prompts
├── install.sh                   # curl install script
├── .github/workflows/release.yml
├── go.mod
└── go.sum
```
