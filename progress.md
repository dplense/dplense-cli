# Development Progress Summary

This document summarizes all the improvements and fixes made during the current development session.

## Date: 2026-01-30

---

## 1. Color Coding for List Drives Output

**File**: `cmd/gdaudit/scan.go`

**Changes**:
- Added color styling to status indicators in `--list-drives` command output
- Green color for "✓ Active" drives
- Red color for "✗ Excluded" drives  
- Yellow color for "○ Not in scope" drives

**Implementation**:
- Imported `github.com/charmbracelet/lipgloss` for terminal styling
- Created color styles using ANSI color codes:
  - Green: `"2"`
  - Red: `"1"`
  - Yellow: `"3"`
- Applied colors to status strings in the `listDrives` function

**Impact**: Makes it easier to visually distinguish between active, excluded, and out-of-scope drives in the output.

---

## 2. Color Coding for WARN Log Messages

**File**: `internal/logger/logger.go`

**Changes**:
- Added yellow color styling to WARN log level messages
- Added red color styling to ERROR log level messages
- DEBUG and INFO messages remain uncolored

**Implementation**:
- Imported `github.com/charmbracelet/lipgloss`
- Modified `writeText()` function to apply color styling based on log level
- Colors are automatically disabled when output is redirected to a file

**Impact**: WARN messages are now more visible and easier to spot in log output.

---

## 3. Fixed Interactive Mode (TUI) Issues

**Files**: 
- `cmd/gdaudit/tui.go`
- `internal/tui/progress.go`
- `internal/tui/model.go`
- `internal/logger/logger.go`

**Problems Fixed**:
1. Logger output interfering with TUI display
2. Progress updates not being sent to TUI
3. Spinner animation not working

**Changes**:

### Logger Suppression (`cmd/gdaudit/tui.go`):
- Created a logger that writes to `io.Discard` to suppress all log output in TUI mode
- Recreated scanner with silent logger inside `launchTUI()` function
- Added `NewWithWriter()` function to logger package for custom output writers

### Progress Updates (`cmd/gdaudit/tui.go`, `internal/tui/model.go`):
- Exported `ScanProgressMsg` type so it can be sent from outside the package
- Set up `scanner.SetProgressFunc()` to send progress updates to TUI program
- Added handling for `ScanProgressMsg` in the main model

### Spinner Animation (`internal/tui/progress.go`):
- Added `tick()` command that sends `tickMsg` every 100ms
- Spinner increments on each tick for smooth animation
- Fixed duplicate spinner increment bug in `View()` function

**Impact**: TUI now displays cleanly without logger interference, shows real-time progress updates, and has a working animated spinner.

---

## 4. Replaced Spinner with Progress Bar

**File**: `internal/tui/progress.go`

**Changes**:
- Replaced spinner animation with a visual progress bar
- Progress bar uses Unicode block characters (█ for filled, ░ for empty)
- Added animated leading edge for visual feedback
- Progress bar fills based on files scanned (linear scale with max estimate of 50,000 files)

**Implementation**:
- Created `renderProgressBar()` function
- Progress bar width: 50 characters
- Color styling: Blue (`69`) for filled portions, Gray (`240`) for empty portions
- Animation updates every 100ms via ticker

**Impact**: Provides better visual feedback during scanning operations.

---

## 5. Auto-Detection of Output Format from Filename

**File**: `cmd/gdaudit/scan.go`

**Problem**: Using `--output json` created a file named "json" with table format instead of JSON format.

**Changes**:
- Added auto-detection of format from output filename
- If `--output` ends with `.json`, automatically uses JSON format
- If `--output` is exactly `json` (without extension), uses JSON format
- Same logic applies for CSV format

**Implementation**:
- Added format detection logic before determining output writer
- Checks file extension first, then checks if filename matches format name
- Only applies when format is still at default "table"

**Impact**: More intuitive behavior - users can specify format via filename without needing `--format` flag.

**Examples**:
- `--output json` → Creates "json" file with JSON format
- `--output results.json` → Creates "results.json" file with JSON format
- `--output data.csv` → Creates "data.csv" file with CSV format

---

## 6. Pattern/Mask Support for --shared-with Flag

**Files**:
- `internal/audit/scanner.go`
- `cmd/gdaudit/scan.go`

**Changes**:
- Added wildcard pattern matching support to `--shared-with` filter
- Supports `*` wildcards for flexible email matching
- Updated help text to document pattern support

**Implementation**:
- Added `matchesSharedWithPattern()` method using `filepath.Match` for glob pattern matching
- Updated `processFile()` to use pattern matching instead of exact string match
- Imported `path/filepath` package for pattern matching

**Supported Patterns**:
- **Exact match**: `user@example.com` - matches exactly this email
- **Domain wildcard**: `*@example.com` - matches all emails from example.com domain
- **Prefix wildcard**: `user*@example.com` - matches emails starting with "user" from example.com
- **Suffix wildcard**: `*user@example.com` - matches emails ending with "user" from example.com
- **Multiple wildcards**: `*user*@example.com` - matches emails containing "user" anywhere

**Impact**: Enables flexible filtering of files shared with multiple users matching a pattern.

**Examples**:
```bash
# Find files shared with any email from competitor.com
gdaudit scan --scope shared-drives --shared-with "*@competitor.com"

# Find files shared with emails starting with "admin" from example.com
gdaudit scan --scope shared-drives --shared-with "admin*@example.com"
```

---

## Summary of Files Modified

1. `cmd/gdaudit/scan.go` - Color coding, format auto-detection, help text updates
2. `cmd/gdaudit/tui.go` - TUI logger suppression, progress callback setup
3. `internal/logger/logger.go` - Color coding for log levels, `NewWithWriter()` function
4. `internal/tui/progress.go` - Progress bar implementation, spinner removal
5. `internal/tui/model.go` - Progress message handling
6. `internal/audit/scanner.go` - Pattern matching for shared-with filter

---

## Testing Status

- ✅ All code compiles successfully
- ✅ All tests pass
- ✅ No linter errors
- ✅ Build successful

---

## Date: 2026-02-24

---

## 7. Fixed Race Condition in Scanner Filters

**Files**:
- `internal/audit/scanner.go`
- `cmd/gdaudit/scan.go`
- `cmd/gdaudit/tui.go`
- `internal/audit/scanner_test.go`

**Problem**: `Scanner.filterSharedWith` and `Scanner.filterPublicOnly` were mutable struct fields set via setter methods, but read concurrently by worker goroutines in `processFile()` without synchronization — a data race.

**Changes**:
- Added `ScanOptions` struct with `FilterSharedWith` and `FilterPublicOnly` fields
- Changed `Scan()` signature to accept `ScanOptions` as a value parameter
- Threaded `opts` through `scanTarget()` → `processFiles()` → `processFile()`
- Removed mutable fields and setter methods (`SetFilterSharedWith`, `SetFilterPublicOnly`)
- Made `matchesSharedWithPattern()` a standalone function (no longer a method)
- Updated `scan.go` and `tui.go` to build `ScanOptions` before calling `Scan()`
- Updated all tests to pass `ScanOptions{}` to `processFile()`

**Impact**: Eliminates race condition. Filter values are immutable during scan execution (passed by value to each goroutine).

**Testing**: `go build ./...` ✅, `go test ./...` ✅, `go vet ./...` ✅

---

## 8. Log Config Load Failures

**File**: `cmd/gdaudit/root.go`

**Problem**: `initConfig()` silently ignored config load errors and fell back to defaults. Users had no way to know their config file was malformed or inaccessible.

**Changes**:
- Added `fmt.Fprintf(os.Stderr, ...)` warning before falling back to defaults
- Uses stderr directly since logger isn't initialized yet at this point (cobra.OnInitialize runs initConfig before initLogger)

**Impact**: Users now see a warning when their config fails to load, instead of silently getting defaults.

**Testing**: `go build ./...` ✅, `go vet ./...` ✅

---

## 9. Extracted Duplicate Credentials Resolution to Helper

**Files**:
- Created `cmd/gdaudit/helpers.go`
- Modified `cmd/gdaudit/scan.go`
- Modified `cmd/gdaudit/revoke.go` (2 occurrences)
- Modified `cmd/gdaudit/tui.go`

**Problem**: Same 12-line credentials resolution logic (resolve path → check file existence → return error) was duplicated 4 times across scan.go, revoke.go (twice), and tui.go.

**Changes**:
- Created `resolveCredentialsPath(cfg, flagOverride)` helper in `helpers.go`
- Replaced all 4 duplicated blocks with single helper call

**Impact**: Single source of truth for credentials resolution. Easier to modify behavior (e.g., add logging, change fallback) in one place.

**Testing**: `go build ./...` ✅, `go vet ./...` ✅, `go test ./...` ✅

---

## 10. Added Tests for revoke/ Package

**Files created**:
- `internal/revoke/file_test.go` (9 tests)
- `internal/revoke/user_test.go` (4 tests)

**Problem**: `internal/revoke/file.go` and `user.go` had zero test coverage. These are the most critical functions — they DELETE permissions.

**Test cases for FileRevoker**:
- External user gets revoked
- Internal user NOT revoked
- Owner permission NOT revoked
- Dry-run mode — no actual deletions
- Public "anyone" permission gets revoked
- Specific user from file gets revoked
- Owner cannot be revoked (returns error)
- User not found — no error, no deletion
- RevokeUserFromFile dry-run

**Test cases for UserRevoker**:
- User revoked from multiple files
- Dry-run mode
- ListPermissions error — continues processing other files
- No matching files — no API calls made

**Coverage**: 89.0% of statements in `internal/revoke/`

**Testing**: `go test ./...` ✅ (13/13 new tests pass)

---

## 11. Passed Context to MetadataResolver Methods

**Files modified**:
- `internal/drive/metadata.go`
- `internal/audit/scanner.go`

**Problem**: `GetFolderInfo()` and `buildPathRecursive()` used `context.Background()` internally, ignoring the caller's context. This meant cancellation signals didn't propagate — if a scan was cancelled, metadata resolution continued making API calls.

**Changes**:
- Added `ctx context.Context` parameter to: `GetDriveName()`, `GetFolderPath()`, `GetFolderName()`, `buildPathRecursive()`, `GetFolderInfo()`
- Removed `context.Background()` calls (lines 160, 199)
- Updated callers in `scanner.go` to pass `ctx` through
- Removed outdated TODO comment about needing context parameter

**Impact**: Cancellation and timeouts now propagate correctly through the entire scan → metadata resolution chain.

**Testing**: `go build ./...` ✅, `go vet ./...` ✅, `go test ./...` ✅

---

## 12. Integrated Rate Limiter into Drive Client

**File modified**: `internal/drive/client.go`

**Problem**: Rate limiter (`pkg/gdrive/ratelimit.go`) was fully implemented and tested but never used. Large scans could hit Google Drive API 429 errors (quota: 1000 requests per 100 seconds).

**Changes**:
- Added `rateLimiter *gdrive.RateLimiter` field to `client` struct
- Instantiated rate limiter in `NewClient()` with sensible defaults (5 retries, 1s base delay, 30s max delay)
- Added `waitForQuota()` helper method
- Added `c.waitForQuota(ctx)` call at the start of all 6 API methods: `ListFiles`, `GetFile`, `ListPermissions`, `DeletePermission`, `ListDrives`, `GetDrive`
- Rate limiter respects context cancellation

**Impact**: API calls are now automatically throttled to stay within Google Drive API quotas. Prevents 429 errors during large scans.

**Testing**: `go build ./...` ✅, `go vet ./...` ✅, `go test ./...` ✅

---

## 13. Verified Binaries Not Tracked in Git

**Check**: `git ls-files -- gdaudit gdrive-audit results.json` returned empty — binaries are not tracked.

**Status**: `.gitignore` already excludes `gdaudit`, `gdrive-audit`, and `*.json`. No action needed.

---

## Final Verification

- `go build ./...` ✅
- `go vet ./...` ✅
- `go test ./...` ✅ (all packages pass)
- `go test -race ./...` ✅ (no race conditions detected)
- `go test -cover ./internal/revoke/...` → **89.0% coverage**

---

## 14. Improved Scan CLI Output

**Files modified**:
- `internal/audit/scanner.go`
- `internal/report/progress.go`
- `cmd/gdaudit/scan.go`

**Problems fixed**:
1. **Bug**: "Scanning target 34/69" showed wrong total — `len(targets)` instead of `len(filteredTargets)` after exclusions
2. **Interleaving**: INFO log lines and progress bar mixed on stderr (both writing unsynchronized)
3. **Noise**: "Processing page 1...", "Scanning target X/Y", "Completed target X/Y" duplicated info from progress bar
4. **Duplicate**: "Scan complete" printed twice — once by logger, once by progress reporter
5. **Progress lacked context**: Just "Scanning... 200 files" — didn't show which drive was being scanned

**Changes**:

### scanner.go:
- Fixed target count bug: `len(targets)` → `len(filteredTargets)` on line 165
- Downgraded per-target and per-page logs from `Info` to `Debug` (lines 165, 185, 241)
- Downgraded final "Scan complete" log to `Debug` (line 196)
- Consolidated pre-scan messages: "Found N targets (X excluded, Y to scan)"
- Added `targetProgressFunc` callback field and `SetTargetProgressFunc()` setter
- Called `targetProgressFunc(idx+1, len(filteredTargets), targetName)` before each target

### progress.go:
- Added `targetIdx`, `targetTotal`, `targetName`, `startTime` fields
- Added `SetTarget(idx, total, name)` method — called before each target
- Changed `Update()` to always render (removed 50-file throttle since `\r` rewrite is cheap)
- New format: `Scanning [12/66] Marketing Drive — 847 files, 23 issues`
- Added duration to `Finish()`: `Scan complete: 2340 files scanned, 87 issues found in 5m23s`
- Added `formatDuration()` helper (ms / s / Xm YYs)
- Truncates long drive names (>40 chars) with "..."

### scan.go:
- Wired `scanner.SetTargetProgressFunc(progressReporter.SetTarget)`
- Removed redundant `appLogger.Info("Starting scan with scope: %s")` (scanner already logs scope info)

**Expected output (clean)**:
```
[21:24:01] INFO: Using credentials file: credentials.json
[21:24:01] INFO: Found 69 targets (3 excluded, 66 to scan)

Scanning [12/66] Marketing Drive — 847 files, 23 issues

Scan complete: 2340 files scanned, 87 issues found in 5m23s
```

**Testing**: `go build ./...` ✅, `go vet ./...` ✅, `go test ./...` ✅

---

## 15. Fixed `--list-drives` Counts & Eliminated Drive Filtering Duplication

**Files modified**:
- `internal/audit/scope.go`
- `internal/audit/scanner.go`
- `cmd/gdaudit/scan.go`

**Problems fixed**:
1. **Wrong count**: `--list-drives` showed "66 active, 3 excluded" when only 1 drive was in `included_drives` — `included_drives` filtering was not reflected in header/footer summary
2. **Duplicate logic**: Include/exclude drive filtering duplicated in 3 places: scanner methods, `isExcluded()` function, and inline whitelist check in `listDrives()` loop

**Changes**:

### scope.go:
- Added exported `ShouldIncludeDrive(target, includedDrives)` and `ShouldExcludeDrive(target, excludedDrives)` standalone functions
- Single source of truth for drive filtering logic

### scanner.go:
- Replaced `s.shouldIncludeDrive(target)` → `ShouldIncludeDrive(target, s.config.IncludedDrives)`
- Replaced `s.shouldExcludeDrive(target)` → `ShouldExcludeDrive(target, s.config.ExcludedDrives)`
- Deleted private methods `shouldIncludeDrive()` and `shouldExcludeDrive()`

### scan.go:
- Deleted `isExcluded()` standalone function
- Replaced inline whitelist check in `listDrives()` with `audit.ShouldIncludeDrive()`
- Pre-loop now counts all 3 categories: `excludedCount`, `outOfScopeCount`, `activeCount`
- Per-row status uses shared functions: `audit.ShouldExcludeDrive()` / `audit.ShouldIncludeDrive()`
- Simplified header to single line: `Found 69 shared drives for scope 'shared-drives':`
- Footer now shows all categories: `Total: 69 shared drives (1 active, 3 excluded, 65 not in scope)`
- Downgraded "Resolving targets" log from Info to Debug
- Downgraded "Found N shared drives" in scope.go from Info to Debug

**Testing**: `go build ./...` ✅, `go vet ./...` ✅, `go test ./...` ✅

---

## 16. Added Animated Spinner to CLI Progress

**Files modified**:
- `internal/report/progress.go` (rewritten)
- `cmd/gdaudit/scan.go` (1 line added)

**New dependency**: `github.com/charmbracelet/bubbles` (spinner component)

**Problem**: Progress line was static between data updates — no visual feedback while waiting for API responses.

**Changes**:

### progress.go (rewritten):
- Replaced `\r`-based manual rendering with a Bubble Tea program
- Uses `bubbles/spinner.Dot` (braille dots: `⣾⣽⣻⢿⡿⣟⣯⣷`) for smooth animation
- New internal `progressModel` Bubble Tea model handles `spinner.TickMsg`, `progressMsg`, `targetMsg`, `doneMsg`
- `ProgressReporter` exported API preserved: `NewProgressReporter()`, `SetTarget()`, `Update()`, `Finish()`
- Added `Start()` method that launches `tea.Program` in a background goroutine
- `Update()` / `SetTarget()` → `program.Send()` (thread-safe)
- `Finish()` → sends `doneMsg`, waits for program exit, prints final summary with duration

### scan.go:
- Added `progressReporter.Start()` call before scan begins

**Output**:
```
⣻ Scanning [1/1] Secure Client Projects — 200 files, 27 issues
```

**Testing**: `go build ./...` ✅, `go vet ./...` ✅, `go test ./...` ✅, `go test -race ./...` ✅

---

## Date: 2026-02-25

---

## 17. CLI Command Improvements (15 items)

Implemented a comprehensive set of improvements based on UX analysis of all CLI commands.

### Bug fixes:

**1.1 Fixed default_scope from config** (`cmd/gdaudit/scan.go`)
- `scanScope` was `""` by default but config's `default_scope` was never applied
- Added: `if scanScope == "" { scanScope = cfg.DefaultScope }`

**1.2 Fixed case-sensitive confirmation** (`cmd/gdaudit/revoke.go`)
- `confirmation != "yes"` rejected "Yes", "YES" etc.
- Changed to `strings.ToLower(strings.TrimSpace(confirmation)) != "yes"` in both `runRevokeFile()` and `runRevokeUser()`

**1.4 Separate revokeInput variable** (`cmd/gdaudit/revoke.go`)
- `revokeUserCmd` was using `reportInput` from report.go — potential conflict
- Created dedicated `revokeInput string` variable

**1.3 Added excel format alias** (`cmd/gdaudit/report.go`)
- `--format excel` now accepted as alias for `xlsx`
- Fixed README which documented non-existent `by-file`, `by-risk` strategies

### UX improvements:

**2.1 Added Quick Start to root help** (`cmd/gdaudit/root.go`)
- Root `--help` now shows workflow: `init → scan → report → revoke`

**2.2 Added workflow hints to help text** (`cmd/gdaudit/revoke.go`, `cmd/gdaudit/report.go`)
- `--input` flag descriptions now mention it expects JSON from `gdaudit scan --output`
- Added `Long` description to `revokeUserCmd` with examples

**2.3 Removed `--dry-run` flag** (`cmd/gdaudit/revoke.go`)
- Dry-run is now always the default; `--confirm` is the only opt-in
- Simpler UX: removed confusing `--dry-run=false` + `--confirm` interaction

**2.4 Added `--risk-level` filter** (`cmd/gdaudit/scan.go`, `internal/audit/scanner.go`)
- New flag: `--risk-level critical,high` filters scan results by risk level
- Added `FilterRiskLevels []string` to `ScanOptions`
- Filter applied in `processFile()` — checks max risk level of external permissions

**2.5 Revoke summary with count** (`cmd/gdaudit/revoke.go`, `internal/revoke/file.go`, `internal/revoke/user.go`)
- `RevokeExternalPermissions()` now returns `(int, error)` — count of revoked permissions
- `RevokeUserFromFiles()` now returns `(revokedCount, filesAffected, error)`
- CLI shows: "Revoked 5 permission(s) from 3 file(s) for user ext@example.com"

**2.6 Risk level breakdown in scan summary** (`internal/report/progress.go`)
- `FinishWithResult(result)` replaces `Finish()` — accepts scan result for risk analysis
- Summary now shows: "Scan complete: 200 files, 27 issues (3 critical, 8 high, 16 medium) in 25s"

**2.7 Auto-generate report output filename** (`cmd/gdaudit/report.go`)
- If `--output` is omitted, auto-generates from `--input`: `scan.json → scan.xlsx`

**2.8 Added `--wide` flag for list-drives** (`cmd/gdaudit/scan.go`)
- `--wide` disables truncation of names, owner info, and IDs in `--list-drives` output

### CI/CD improvements:

**3.1 Environment variable support** (`cmd/gdaudit/root.go`)
- `GDAUDIT_CREDENTIALS` — path to credentials file
- `GDAUDIT_IMPERSONATE` — user email for delegation
- `GDAUDIT_DEBUG=1` — enable debug logging
- Priority: config file < env vars < CLI flags

**3.2 Added `--quiet` flag** (`cmd/gdaudit/root.go`)
- `-q` / `--quiet` suppresses all log output (writes to `io.Discard`)
- Useful for `gdaudit scan --format json -q | jq` workflows

### Documentation:
- Updated README.MD to match actual code: corrected report strategies, revoke flags, added examples

**Testing**: `go build ./...` ✅, `go vet ./...` ✅, `go test ./...` ✅

---

## Date: 2026-03-06

---

## 18. Fixed Duplicated Output & Spinner Mixing

**Files**: `internal/report/progress.go`, `internal/audit/scanner.go`, `internal/audit/scope.go`

**Problems fixed**:
1. **Duplicated summary**: `FinishWithResult()` printed "Scan complete: X files, Y issues..." AND the table's summary box showed the same info
2. **Spinner/logger mixing**: `logger.Print` wrote to stderr simultaneously with spinner, causing text concatenation on the same line
3. **Race condition**: `fmt.Fprint` in spinner goroutine was outside mutex

**Changes**:
- Renamed `FinishWithResult(result)` → `Stop()` — removed text summary (table summary box is sufficient)
- Changed `logger.Print` → `logger.Info` in `scanner.go` and `scope.go` to avoid mixing with spinner
- Moved `fmt.Fprint` inside mutex in spinner goroutine
- Updated caller in `scan.go`: `progressReporter.FinishWithResult(result)` → `progressReporter.Stop()`

**Testing**: `go build ./...` ✅, `go test ./...` ✅

---

## 19. Fixed Default Scope Config Override

**File**: `~/.gdaudit/config.yaml`

**Problem**: Config file had `default_scope: active` which overrides the code default, causing 401 errors when scanning without explicit `--scope` flag.

**Change**: `default_scope: active` → `default_scope: shared-drives`

---

## 20. Fixed user: Scope 401 Error

**File**: `internal/audit/scope.go`

**Problem**: `resolveUser()` required Directory API (admin.directory.user.readonly scope) which returned 401 when scope not authorized. The function hard-failed instead of falling back.

**Changes**:
- `resolveUser` now gracefully falls back: tries Directory API first, if fails creates Target from email directly
- Changed log level from `Warn` to `Info` for Directory API fallback message (avoids scary output mixed with spinner)

**Testing**: `go build ./...` ✅, `go test ./...` ✅

---

## 21. Unified --filter Flag

**Files**: `cmd/gdaudit/scan.go`

**Problem**: Three separate flags (`--shared-with`, `--public`, `--risk-level`) were verbose and inconsistent.

**Changes**:
- Replaced three flags with single `--filter` using key:value syntax (repeatable via `StringArrayVar`)
- Added `parseFilters()` function that converts filter strings to `ScanOptions`
- Updated help text with syntax and examples

**Supported filters**:
- `--filter shared-with:*@gmail.com` — files shared with email/pattern
- `--filter public` — only "Anyone with link" files
- `--filter risk:critical,high` — only issues at given risk levels

**Examples**:
```bash
gdaudit scan --filter shared-with:*@gmail.com
gdaudit scan --filter public --filter risk:critical
```

**Testing**: `go build ./...` ✅, `go test ./...` ✅

---

## 22. Classification Labels in Scan Output

**Files** (11 files modified/created):

**Problem**: No visibility into Google Drive document classification labels. Users couldn't see which files were labeled (e.g., "Confidential") and which were unclassified.

**Architecture**:
- Drive Labels API (`drivelabels/v2`) fetches all published organization labels
- Label Resolver maps opaque label/field/choice IDs to human-readable names
- `ListFiles` conditionally includes `labelInfo` when labels are configured
- Three-state label field: actual name / "Unclassified" / "Cannot be retrieved"

**Changes**:

| File | Change |
|------|--------|
| `internal/auth/scopes.go` | Added `drive.labels.readonly` to `RequiredScopes`, new `LabelsScopes()` |
| `internal/auth/service_account.go` | Added `NewLabelsService()` for Drive Labels API |
| `internal/labels/resolver.go` | **New file**: fetches org labels, maps IDs to names, resolves selection fields |
| `pkg/gdrive/client.go` | Added `Labels []string` to `File`, `SetIncludeLabels` to `DriveClient` interface |
| `internal/drive/client.go` | Added `includeLabels`, `labelResolver`, label parsing in `ListFiles`, `ConfigureLabelResolver` helper |
| `pkg/models/scan.go` | Added `Label string` to `FileIssue` |
| `internal/audit/scanner.go` | Added `labelsAvailable` flag and `SetLabelsAvailable()`, sets Label on each issue |
| `cmd/gdaudit/scan.go` | Initializes label resolver with graceful fallback, wires up to drive client |
| `internal/report/table.go` | Added `Label:` row in file output |
| `internal/report/csv.go` | Added `Label` column after `Drive Name` |
| `pkg/gdrive/mocks/drive_mock.go` | Added `SetIncludeLabels` no-op for interface compliance |

**Graceful degradation**: If Labels API scope not authorized in domain-wide delegation:
- `labels.NewResolver()` fails → labels not configured
- Scan runs normally without labels API calls
- Label field shows `"Cannot be retrieved"` in output

**Three label states**:
1. **Label name** (e.g., "Confidential") — file has label, Labels API available
2. **"Unclassified"** — Labels API available but file has no labels
3. **"Cannot be retrieved"** — Labels API scope not configured

**Testing**: `go build ./...` ✅, `go test ./...` ✅

---

## 23. Fixed 401 Auth Error from Labels Scope in DriveScopes

**File**: `internal/auth/scopes.go`

**Problem**: `drive.labels.readonly` was mistakenly added to `DriveScopes()`, which is used by `NewDriveService()`. Since Google Admin hadn't authorized this scope for domain-wide delegation, the **entire Drive service** failed with 401 "unauthorized_client" — breaking all scanning.

**Change**: Removed `drive.labels.readonly` from `DriveScopes()`. Scope remains in:
- `RequiredScopes` — for documentation/instructions to admins
- `LabelsScopes()` — used only by `NewLabelsService()` (separate auth token)

**Impact**: Drive service now requests only 2 scopes (`drive.readonly`, `drive.metadata.readonly`). Labels API uses its own separate auth token with `drive.labels.readonly`.

**Testing**: `go build ./...` ✅, `go test ./...` ✅
