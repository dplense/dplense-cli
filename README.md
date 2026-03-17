# DPlense CLI

**dplense** is a multi-cloud security audit CLI tool. It scans cloud storage for external file shares, public links, and permission risks across Google Drive, Microsoft 365, and Slack.

## Features

- **Multi-Provider**: Google Drive, Microsoft 365 (SharePoint + OneDrive), Slack
- **Scanning**: Scan shared drives, active/suspended users, or specific accounts
- **Filtering**: Filter by email patterns (wildcards), public links, risk levels
- **Multiple Output Formats**: Table, JSON, CSV, Excel reports
- **Interactive TUI**: Terminal UI for browsing issues, viewing details, and revoking permissions
- **Permission Revocation**: Safely revoke external permissions with dry-run by default
- **Configurable**: YAML-based configuration with domain whitelisting/blacklisting
- **Progress Tracking**: Real-time progress with ETA and per-drive statistics

## Installation

### Prerequisites

- Go 1.25 or later
- Provider-specific credentials (see Quick Start below)

### Install with Go

```bash
go install github.com/dplense/dplense-cli/cmd/dplense@latest
```

### Build from Source

```bash
git clone https://github.com/dplense/dplense-cli.git
cd dplense-cli
go build -o dplense ./cmd/dplense
```

## Quick Start

### 1. Initialize Configuration

Run the interactive setup wizard:

```bash
dplense init
```

This will:
- Create configuration directory (`~/.dplense/`)
- Guide you through setting up credentials
- Create a default configuration file

### 2. Configure Google Workspace

1. **Create a Service Account**:
   - Go to [Google Cloud Console](https://console.cloud.google.com/)
   - Create a new project or select existing one
   - Enable Google Drive API and Admin SDK API
   - Create a service account and download JSON credentials

2. **Configure Domain-Wide Delegation**:
   - Go to [Google Admin Console](https://admin.google.com)
   - Navigate to Security > API Controls > Domain-wide Delegation
   - Add your service account with these scopes:
     - `https://www.googleapis.com/auth/drive.readonly`
     - `https://www.googleapis.com/auth/drive.metadata.readonly`
     - `https://www.googleapis.com/auth/admin.directory.user.readonly` (for user enumeration)

3. **Place Credentials**:
   ```bash
   cp /path/to/credentials.json ~/.dplense/credentials.json
   ```

### 3. Run Your First Scan

```bash
# Scan shared drives
dplense scan --scope shared-drives

# Scan with impersonation (required for domain-wide delegation)
dplense scan --scope shared-drives --impersonate admin@yourdomain.com
```

## Configuration

Configuration file is located at `~/.dplense/config.yaml`. Example:

```yaml
# Internal domains (not considered external)
internal_domains:
  - yourdomain.com
  - subsidiary.com

# Trusted domains (lower risk)
trusted_domains:
  - partner.com

# Google-specific settings
google:
  credentials_path: ~/.dplense/credentials.json
  impersonate_user: admin@yourdomain.com
  included_drives: []   # Empty = all drives
  excluded_drives:
    - "Archive Drive"

# Default settings
default_scope: shared-drives
dry_run: true  # Safety default for revoke operations
```

### Environment Variables

| Variable | Description |
|----------|-------------|
| `DPLENSE_CREDENTIALS` | Path to Google credentials file |
| `DPLENSE_IMPERSONATE` | User email for domain-wide delegation |
| `DPLENSE_DEBUG` | Enable debug logging (`1` or `true`) |
| `DPLENSE_MS_TENANT_ID` | Microsoft 365 tenant ID |
| `DPLENSE_MS_CLIENT_ID` | Microsoft 365 client ID |
| `DPLENSE_MS_CLIENT_SECRET` | Microsoft 365 client secret |
| `DPLENSE_SLACK_BOT_TOKEN` | Slack bot token |

## Usage

### Scan Command

Scan cloud storage for security issues.

```bash
dplense scan [flags]
```

#### Providers

- `google` (default) — Google Drive
- `microsoft` — SharePoint + OneDrive
- `slack` — Slack files and channels

#### Scopes

- `shared-drives` — Scan all shared drives (Google)
- `active` — Scan all active users
- `suspended` — Scan all suspended users
- `user:<email>` — Scan specific user
- `sharepoint` — Scan SharePoint sites (Microsoft)
- `onedrive` — Scan OneDrive (Microsoft)
- `all` — Scan everything (Slack)

#### Flags

| Flag | Description | Example |
|------|-------------|---------|
| `--provider` | Cloud provider | `--provider microsoft` |
| `--scope` | Scan scope | `--scope shared-drives` |
| `--filter` | Filter results (repeatable) | `--filter shared-with:*@gmail.com` |
| `--interactive`, `-i` | Launch interactive TUI | `-i` |
| `--format` | Output format: `table`, `json`, `csv` | `--format json` |
| `--output` | Write results to file | `--output results.json` |
| `--list-drives` | List all drives without scanning | `--list-drives` |
| `--credentials` | Path to credentials file | `--credentials ./creds.json` |
| `--impersonate` | User email for delegation | `--impersonate admin@domain.com` |

#### Filter Syntax

Filters can be combined with multiple `--filter` flags:

- `--filter shared-with:*@gmail.com` — Files shared with email/pattern
- `--filter public` — Only files with "Anyone with link" access
- `--filter risk:critical,high` — Only issues at given risk levels

#### Examples

```bash
# Basic scan of shared drives
dplense scan

# Scan Microsoft 365 SharePoint
dplense scan --provider microsoft --scope sharepoint

# Scan Slack
dplense scan --provider slack --scope all

# Output to JSON file
dplense scan --output results.json

# Find files shared with specific domain
dplense scan --filter shared-with:*@competitor.com

# Find public files with critical risk
dplense scan --filter public --filter risk:critical

# Interactive TUI mode
dplense scan -i

# List drives without scanning
dplense scan --list-drives

# Scan specific user
dplense scan --scope user:john@example.com
```

### Revoke Command

Revoke external permissions from files.

```bash
# Revoke permissions from a specific file
dplense revoke file <file_id> [flags]

# Revoke a user's access from all files
dplense revoke user <email> [flags]
```

#### Flags

| Flag | Description |
|------|-------------|
| `--confirm` | Actually apply changes (default is dry-run/preview) |
| `--user` | Revoke only this specific user (for `revoke file`) |
| `--input` | Path to JSON from `dplense scan --output` (for `revoke user`) |

#### Examples

```bash
# Preview revoking all external permissions from a file
dplense revoke file 1BxiMVs0Xzy5dD1KZzJz

# Actually revoke (requires --confirm)
dplense revoke file 1BxiMVs0Xzy5dD1KZzJz --confirm

# Revoke user from all files in scan results
dplense revoke user external@competitor.com --input results.json --confirm
```

### Report Command

Generate Excel reports from scan results.

```bash
dplense report --input results.json
dplense report --input results.json --strategy by-owner --output report.xlsx
```

### Init Command

Interactive wizard to set up configuration and credentials.

```bash
dplense init
```

## Interactive TUI

Launch with `dplense scan -i`:

- Browse files with external permissions
- View detailed file info, internal users, and external shares
- Revoke permissions directly from the TUI
- Filter by file/drive name
- Keyboard: `↑↓` navigate, `Enter` open, `/` search, `r` revoke, `o` open in browser, `q` quit

## Risk Levels

| Level | Color | Description |
|-------|-------|-------------|
| **Critical** | Red | Public "Anyone with Link" permissions |
| **High** | Orange | External users with write/organizer access |
| **Medium** | Yellow | External users with read/comment access |
| **Low** | Green | Internal users |

## Troubleshooting

### Authentication Errors

**Error**: `unauthorized_client` or `401`

1. Verify domain-wide delegation is configured in Google Admin Console
2. Ensure service account has correct scopes authorized
3. Check that `--impersonate` flag is set with a valid admin user
4. Wait a few minutes after configuring delegation for changes to propagate

### No Files Found

1. Scope doesn't match any targets (check with `--list-drives`)
2. All files are filtered out by internal/trusted domain settings
3. Drive access restrictions (check service account permissions)

### Permission Denied Errors

**Error**: `403: The attempted action requires shared drive membership`

Ensure the service account has access to the shared drives. This may require adding the service account as a member of the shared drive.

## Contributing

Contributions are welcome! Please:

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Support

- [GitHub Issues](https://github.com/dplense/dplense-cli/issues)
- [GitHub Repository](https://github.com/dplense/dplense-cli)
