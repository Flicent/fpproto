# Changelog

## v1.0.0

Initial release of fpproto.

- `fpproto setup` — one-time machine configuration, pulls org credentials via GitHub CLI
- `fpproto create <name>` — provisions a full prototype environment (Supabase project, GitHub repo from template, Vercel deployment, local clone with `.env.local`)
- `fpproto clone <name>` — clones an existing prototype onto the current machine with credentials and dependencies
- `fpproto destroy <name>` — archives a prototype with confirmation prompt, tears down Supabase and Vercel infrastructure
- `fpproto list` — displays a styled table of all active prototypes in the org
- `fpproto update` — self-updates the CLI binary from GitHub Releases
- Background version nudge on every command when a newer release is available
- Auto-updating org config when a newer version is published to the config repo
- Styled terminal UI with spinners, color-coded status icons, bordered summary boxes, and formatted tables
- curl-based install script for macOS (ARM64 and Intel)
- GitHub Actions CI/CD with auto-versioning and cross-compiled release builds
