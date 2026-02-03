# GDrive Audit

**gdaudit** CLI tool for auditing Google Drive permissions. It helps to analyze and remediate external file shares across Google Workspace.

## Features

- **Scanning**: Scan active users, suspended users, shared drives, or specific users
- **Filtering**: Filter by external email patterns (with wildcard support), public links, or specific domains
- **Multiple Output Formats**: Generate reports in table, JSON, CSV, or Excel formats
- **Interactive TUI**: EXPEREMENTAL!! Terminal user interface with real-time progress and detailed file views
- **Permission Revocation**: Safely revoke external permissions from files or users
- **Configurable**: YAML-based configuration with domain whitelisting/blacklisting
- **Color-Coded Output**: Visual indicators for status, risk levels, and log messages
- **Progress Tracking**: Real-time progress bars and detailed scanning statistics

## Installation

### Prerequisites

- Go 1.19 or later
- Google Workspace Admin access
- Service Account with Domain-Wide Delegation configured

### Build from Source

```bash
git clone https://gitlab.da.local/admins/gdrive-audit.git
cd gdrive-audit
go build -o gdaudit ./cmd/gdaudit
```

### Binary Installation

Download the latest release binary for your platform and add it to your PATH.

## Quick Start

### 1. Initialize Configuration

Run the interactive setup wizard:

```bash
gdaudit init
```

This will:
- Create configuration directory (`~/.gdaudit/`)
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
   cp /path/to/credentials.json ~/.gdaudit/credentials.json
   ```

### 3. Run Your First Scan

```bash
# Scan shared drives
gdaudit scan --scope shared-drives

# Scan with impersonation (required for domain-wide delegation)
gdaudit scan --scope shared-drives --impersonate admin@yourdomain.com
```

## Configuration

Configuration file is located at `~/.gdaudit/config.yaml`. Example:

```yaml
# Internal domains (not considered external)
internal_domains:
  - yourdomain.com
  - trusted-partner.com

# Trusted domains (lower risk)
trusted_domains:
  - partner.com

# Drive filtering
included_drives: []  # Empty = all drives, or specify drive names/IDs
excluded_drives:
  - "Archive Drive"
  - "0AA4VQp02CjVdUk9PVA"

# Default settings
default_scope: shared-drives
dry_run: true  # Safety default
credentials_path: ~/.gdaudit/credentials.json
impersonate_user: admin@yourdomain.com
```

## Usage

### Scan Command

Scan Google Drive for security issues.

```bash
gdaudit scan [flags]
```

#### Scopes

- `active` - Scan all active users
- `suspended` - Scan all suspended users
- `shared-drives` - Scan all shared drives
- `user:<email>` - Scan specific user (e.g., `user:john@example.com`)

#### Flags

| Flag | Description | Example |
|------|-------------|---------|
| `--scope` | Scan scope | `--scope shared-drives` |
| `--shared-with` | Filter by email pattern (supports wildcards) | `--shared-with "*@competitor.com"` |
| `--public` | Only show files with public "Anyone with Link" permissions | `--public` |
| `--interactive`, `-i` | Launch interactive TUI dashboard | `--interactive` |
| `--format` | Output format: `table`, `json`, or `csv` | `--format json` |
| `--output` | Write results to file (auto-detects format from extension) | `--output results.json` |
| `--list-drives` | List all drives/targets without scanning | `--list-drives` |
| `--credentials` | Path to service account JSON file | `--credentials ./creds.json` |
| `--impersonate` | User email for domain-wide delegation | `--impersonate admin@domain.com` |
| `--debug` | Enable debug logging | `--debug` |
| `--verbose`, `-v` | Enable verbose output | `--verbose` |

#### Examples

```bash
# Basic scan of shared drives
gdaudit scan --scope shared-drives

# Scan and output to JSON file
gdaudit scan --scope shared-drives --output results.json

# Find files shared with specific domain
gdaudit scan --scope shared-drives --shared-with "*@competitor.com"

# Find files shared with users matching pattern
gdaudit scan --scope shared-drives --shared-with "admin*@example.com"

# Find only public files
gdaudit scan --scope shared-drives --public

# Interactive TUI mode
gdaudit scan --scope shared-drives --interactive

# List drives without scanning
gdaudit scan --scope shared-drives --list-drives

# Scan specific user
gdaudit scan --scope user:john@example.com

# Scan with custom credentials
gdaudit scan --scope shared-drives --credentials ./my-creds.json --impersonate admin@domain.com
```

### Revoke Command

Revoke external permissions from Google Drive files.

```bash
# Revoke permissions from a specific file
gdaudit revoke file <file_id> [flags]

# Revoke a user's access from all files
gdaudit revoke user <email> [flags]
```

#### Flags

| Flag | Description |
|------|-------------|
| `--dry-run` | Preview changes without applying them (default: true) |
| `--confirm` | Actually apply changes (required for real revocations) |

#### Examples

```bash
# Preview revoking user from a file
gdaudit revoke file 1BxiMVs0Xzy5dD1KZzJz --dry-run

# Actually revoke user from a file
gdaudit revoke file 1BxiMVs0Xzy5dD1KZzJz --confirm

# Preview revoking user from all files
gdaudit revoke user external@competitor.com --dry-run

# Actually revoke user from all files
gdaudit revoke user external@competitor.com --confirm
```

### Report Command

Generate reports from previous scan results.

```bash
gdaudit report [flags]
```

#### Flags

| Flag | Description | Example |
|------|-------------|---------|
| `--input` | Input JSON file from previous scan | `--input scan-results.json` |
| `--format` | Output format: `excel`, `csv`, `table` | `--format excel` |
| `--output` | Output file path | `--output report.xlsx` |
| `--strategy` | Excel grouping strategy: `by-file`, `by-user`, `by-risk` | `--strategy by-risk` |

#### Examples

```bash
# Generate Excel report
gdaudit report --input results.json --format excel --output report.xlsx

# Generate CSV report
gdaudit report --input results.json --format csv --output report.csv

# Generate Excel report grouped by risk level
gdaudit report --input results.json --format excel --strategy by-risk --output report.xlsx
```

### Init Command

Interactive wizard to set up configuration and credentials.

```bash
gdaudit init
```

This command guides you through:
- Creating configuration directory
- Setting up credentials file path
- Configuring default scope
- Setting up internal/trusted domains

## Output Formats

### Table Format (Default)

Human-readable table output with color-coded risk levels:
- 🔴 **Critical** - Public files or high-risk external shares
- 🟠 **High** - External shares with write access
- 🟡 **Medium** - External shares with comment access
- 🟢 **Low** - External shares with read-only access

### JSON Format

Structured JSON output suitable for automation and integration:

```json
{
  "metadata": {
    "scope": "shared-drives",
    "total_files_scanned": 1234,
    "issues_found": 56,
    "scan_duration": "5m23s"
  },
  "issues": [
    {
      "file_id": "...",
      "file_name": "...",
      "owner_email": "...",
      "permissions": [...]
    }
  ]
}
```

### CSV Format

Comma-separated values for spreadsheet import.

### Excel Format

Rich Excel reports with:
- Color-coded risk levels
- Multiple sheets (by file, by user, by risk)
- Formatted cells and headers

## Pattern Matching

The `--shared-with` flag supports wildcard patterns:

- `*@example.com` - Matches all emails from example.com domain
- `user*@example.com` - Matches emails starting with "user" from example.com
- `*user@example.com` - Matches emails ending with "user" from example.com
- `*user*@example.com` - Matches emails containing "user" anywhere
- `exact@example.com` - Exact email match

## Interactive TUI

Launch the interactive terminal user interface:

```bash
gdaudit scan --scope shared-drives --interactive
```

Features:
- Real-time progress bar
- Live file scanning statistics
- Navigable list of security issues
- Detailed file permission views
- Color-coded risk levels
- Keyboard navigation (arrow keys, Enter, Esc)

## Risk Levels

Files are categorized by risk level based on permission type and access level:

- **Critical**: Public "Anyone with Link" permissions
- **High**: External users with "Writer" or "Owner" role
- **Medium**: External users with "Commenter" role
- **Low**: External users with "Reader" role

## Troubleshooting

### Authentication Errors

**Error**: `unauthorized_client` or `401`

**Solution**:
1. Verify domain-wide delegation is configured in Google Admin Console
2. Ensure service account has correct scopes authorized
3. Check that `--impersonate` flag is set with a valid admin user
4. Wait a few minutes after configuring delegation for changes to propagate

### No Files Found

**Possible causes**:
1. Scope doesn't match any targets (check with `--list-drives`)
2. All files are filtered out by internal/trusted domain settings
3. Drive access restrictions (check service account permissions)

### Permission Denied Errors

**Error**: `403: The attempted action requires shared drive membership`

**Solution**: Ensure the service account has access to the shared drives you're trying to scan. This may require adding the service account as a member of the shared drive.

## Configuration Reference

### Configuration File Location

Default: `~/.gdaudit/config.yaml`

Override with: `--config /path/to/config.yaml`

### Configuration Options

| Option | Type | Description | Default |
|--------|------|-------------|---------|
| `internal_domains` | `[]string` | Domains not considered external | `[]` |
| `trusted_domains` | `[]string` | Domains with lower risk scoring | `[]` |
| `included_drives` | `[]string` | Whitelist of drives to scan (empty = all) | `[]` |
| `excluded_drives` | `[]string` | Blacklist of drives to skip | `[]` |
| `default_scope` | `string` | Default scan scope | `active` |
| `dry_run` | `bool` | Safety default for revoke operations | `true` |
| `credentials_path` | `string` | Path to service account JSON | `~/.gdaudit/credentials.json` |
| `impersonate_user` | `string` | Default user for impersonation | `""` |

## Examples

### Example 1: Find All External Shares

```bash
gdaudit scan --scope shared-drives --output external-shares.json
```

### Example 2: Find Files Shared with Competitor Domain

```bash
gdaudit scan --scope shared-drives \
  --shared-with "*@competitor.com" \
  --output competitor-shares.json
```

### Example 3: Generate Excel Report

```bash
# Scan and save results
gdaudit scan --scope shared-drives --output scan-results.json

# Generate Excel report
gdaudit report --input scan-results.json \
  --format excel \
  --strategy by-risk \
  --output security-report.xlsx
```

### Example 4: Interactive Audit

```bash
gdaudit scan --scope shared-drives --interactive
```

### Example 5: Revoke External User Access

```bash
# First, scan to find files
gdaudit scan --scope shared-drives --shared-with "external@competitor.com" --output results.json

# Preview revocation
gdaudit revoke user external@competitor.com --dry-run

# Actually revoke (requires --confirm)
gdaudit revoke user external@competitor.com --confirm
```

## Contributing

Contributions are welcome! Please:

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a merge request

## License

[Add your license here]

## Support

For issues, questions, or contributions:
- Issue Tracker: [GitLab Issues](https://gitlab.da.local/admins/gdrive-audit/-/issues)
- Repository: [GitLab Repository](https://gitlab.da.local/admins/gdrive-audit)

## Version

Current version: **0.1.0**
