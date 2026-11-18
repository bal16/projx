# TUI Project Info Improvements — Design

**Date:** 2026-11-18
**Status:** Proposed (awaiting approval)
**Scope:** `projx` (Go CLI/TUI, SQLite, gitinfo, plugins unaffected except data shape compatibility noted)

## Intent & Why

Make the TUI project picker/preview give an at-a-glance, actionable view of each project’s current state: when its files were last modified, a complete git working-tree breakdown (not just total dirty count), and (optionally, for GitHub remotes) cached open PR/issue counts. Add an explicit way to refresh status for all projects so cached info can be brought up to date without leaving the TUI.

## Goals

- Show **last modified time** of the project (filesystem mtime of the project root path), computed live in the preview.
- Show **current branch** plus a **complete git status breakdown** (staged, modified, untracked, deleted, renamed, conflicted/unmerged) derived from `git status --porcelain`, cached like today.
- Show **open PR and open issue counts** for GitHub-backed projects, **cached** and **optional per project** (value may be `null` when not fetched/disabled/not applicable).
- Fetch GitHub info on **add, edit-save, and per-project refresh** when enabled for that project and the remote is GitHub; degrade gracefully (no `gh`, no auth, no network, non-GitHub) by leaving cached values unchanged/null.
- Provide a **trigger to refresh status for all projects** (git + GitHub where applicable) from the TUI and keep CLI `pjx refresh` consistent.
- Preserve existing behavior where not mentioned (favorites, tags, open flow, list formats, plugins’ project opening).

## Non-goals

- Live/network fetch on every selection (PR/issue are cached; mtime/git display uses cached git + live mtime only).
- Replacing `gh` CLI with direct GitHub API/token management in this change.
- Showing PR/issue details (titles/URLs) or last-commit message/time (only counts requested).
- Reworking plugin UIs (Noctalia/DMS); they continue to consume existing list/open behavior. Export/import schema is extended compatibly (new nullable fields).

## Anti-goals

- Making the TUI block or fail when GitHub info cannot be fetched.
- Expensive recursive scans to compute “last modified” (no tree walk).

## Constraints

- SQLite schema currently v2; changes require an idempotent migration (v3).
- `model.Project` JSON tags define export/import format; new fields must be nullable/omittable-friendly and keep `pjx import-file` compatible.
- GitHub detection/fetch depends on git remote (typically `origin`) resolving to `github.com`; `gh` CLI must be used when fetching (graceful skip if unavailable or unauthenticated).
- TUI preview has limited width/height; display must stay compact and readable.

## Approaches Considered

### 1. Explicit columns + `gh` CLI + live mtime (chosen)
- **Storage:** additive columns on `projects` for git breakdown (`git_staged`, `git_modified`, `git_untracked`, `git_deleted`, `git_renamed`, `git_conflicted`) and GitHub cache (`gh_enabled` nullable per-project flag, `gh_open_prs`, `gh_open_issues`, `gh_fetched_at` nullable). Keep `git_dirty` as a derived total for compatibility with existing displays/logic, updated alongside breakdown.
- **Git:** extend `gitinfo.Inspect` to parse full porcelain XY codes and return counts (plus branch/IsRepo). Update `Repository.UpdateGitStatus` (and scan/add/list/refresh paths) to persist breakdown.
- **GitHub:** new helper (e.g. `internal/ghinfo`) to (a) resolve GitHub `owner/repo` from `git remote get-url origin` and (b) run `gh pr list -R <repo> --state open --json number --jq 'length'` and `gh issue list ...` (or equivalent) returning nullable counts; invoked only when `gh_enabled` is true for the project and repo is GitHub. On failure/absence, return no update (preserve prior/null) without erroring the caller flow.
- **Mtime:** `os.Stat(project.Path).ModTime()` at render time in the TUI preview (and not stored).
- **Pros:** consistent with current simple typed schema/caching; easy to query/debug; no token handling. **Cons:** schema change per new field (acceptable). **Chosen** for simplicity and alignment.

### 2. JSON blobs for extensible status
- Store `git_status`/`github_info` as JSON text columns. **Pros:** flexible. **Cons:** less explicit, harder to reason/migrate partially, diverges from current column style. Rejected as unnecessary complexity.

### 3. Direct GitHub REST API
- Call API with user token. **Pros:** no `gh` dependency. **Cons:** auth/token/rate-limit handling and security surface, heavier than needed for an optional cached count. Rejected in favor of `gh` CLI with graceful skip.

## Agreed Design Details

### Data model & schema (v3)
- Add columns (idempotent `ALTER TABLE`, bump `schema_version` to 3):
  - `git_staged INTEGER NOT NULL DEFAULT 0`
  - `git_modified INTEGER NOT NULL DEFAULT 0`
  - `git_untracked INTEGER NOT NULL DEFAULT 0`
  - `git_deleted INTEGER NOT NULL DEFAULT 0`
  - `git_renamed INTEGER NOT NULL DEFAULT 0`
  - `git_conflicted INTEGER NOT NULL DEFAULT 0`
  - `gh_enabled INTEGER NULL` — per-project opt-in: `NULL` unset/disabled, `1` enabled, `0` explicitly disabled (treated as off). (Tri-state allows “optional, can be null”.)
  - `gh_open_prs INTEGER NULL`
  - `gh_open_issues INTEGER NULL`
  - `gh_fetched_at INTEGER NULL` (unix seconds)
- `git_dirty` remains (`NOT NULL DEFAULT 0`) = `staged+modified+untracked+deleted+renamed+conflicted` for compatibility.
- `model.Project` gains corresponding fields with nullable pointers/zero-values as appropriate and JSON tags (omitempty where null/empty) so export/import remains compatible. `scanProject`, `Add`, `Update`, `List/Get` read/write the new columns.

### Git status parsing (complete)
Extend `gitinfo.Info` and parsing from `git status --porcelain` (v1) per line XY:
- **Conflicted/unmerged:** any of `U` in X/Y, or `AA`, `DD` → `Conflicted++`.
- **Untracked:** `??` → `Untracked++`.
- **Renamed/copied (staged):** `R*` or `C*` in X → `Renamed++` (counted as staged rename/copy).
- **Staged:** X in `{M,A,D,R,C}` (and not unmerged/untracked) → `Staged++`.
- **Unstaged modified:** Y == `M` → `Modified++`.
- **Unstaged deleted:** Y == `D` → `Deleted++`.
(Each line counted in the most specific applicable buckets so totals reflect a complete breakdown; `Dirty`/total = sum of all buckets.) Keep `IsRepo`/`Branch` behavior. Update `gitinfo` tests for breakdown cases.

### GitHub info (optional per project, cached)
- **Detection:** from `git -C <path> remote get-url origin` (fallback: first remote) parse `github.com[:/]<owner>/<repo>` (strip `.git`). Non-GitHub → not applicable.
- **Fetch:** when `gh_enabled == true` and repo detected, run `gh` commands with a short timeout; parse counts. On any failure (missing `gh`, exit error, parse error) do not fail the parent operation; leave existing cached values/null as-is (or set fetched_at only on success). `gh_fetched_at` records last successful fetch.
- **Triggers:** on `pjx add` and on project edit/save (CLI `pjx set` paths that modify the project / TUI edit save) and on per-project refresh — perform git status refresh and, if enabled, GitHub fetch for that project. `pjx refresh` (all or specific) refreshes git for targets and GitHub for targets where enabled.
- **Config/CLI:** per-project flag only (per request). CLI: `pjx add --github-info` / `--no-github-info` (and `pjx set --github-info/--no-github-info` or equivalent) to set `gh_enabled`; default unset (`null`) = off. TUI edit form exposes the tri-state/boolean choice.

### TUI changes
- **Preview panel** (selected project): show
  - `Modified: <YYYY-MM-DD HH:MM>` (or `-` if path missing/stat fails) from live mtime.
  - `Git: <branch>  S:<n> M:<n> ?:<n> D:<n> R:<n> U:<n>` showing non-zero (or a compact full set) — complete breakdown; omit line if not a repo/no cached branch as today.
  - `PR: <n>  Issues: <n>` when `gh_enabled` is true *and* cached values present; show `PR/Issues: -` (or omit) when enabled but null/unfetched; omit entirely when disabled/null-not-enabled. `Remote` host line retained if set.
  - Existing fields (Name/Path/Editor/Profile/Tags/Group/Opened) retained.
- **Refresh triggers:**
  - `r`: keep existing behavior (reload project list from DB).
  - `R` (new): **refresh status for all projects** — run git (and GitHub where enabled) refresh for all projects, then reload list (with progress/status message, non-blocking via command where appropriate). This satisfies “trigger refresh status semua project”.
  - `:` command `refresh [project]` continues to invoke CLI refresh; palette already suggests `refresh`.
- **Help/status bar:** update hints to include `R: refresh status` (all).
- **Responsive/breakpoints (explicit):**
  - **Two-column** when `width >= 60` and preview has room (current logic): list width ~2/3, min ~38 cols for preview, **capped at 90 cols** so very wide terminals give space to preview. Preview values are truncated to panel width; git/PR/Modified use compact forms (prefer non-zero counts, e.g. `M:3 ?:2`) to fit.
  - **Narrow/single-column** (`width < 60` or insufficient preview room): keep the current list-first rendering; the list row shows a compact git summary so key status remains visible. Full preview is available in two-column widths (no new toggle in this change; call made for scope).
  - **Small height**: content lines = `maxLines` (min 5); prioritize key preview fields first (Name/Path, Git, Modified, PR/Issues) so they remain visible when space is tight.

### CLI changes
- `pjx refresh [project...]` (no args = all): for each target, update git breakdown; if `gh_enabled` true, attempt GitHub fetch and store counts/fetched_at. Report per project as today (extend output to reflect breakdown succinctly if desired).
- `pjx add`: accept `--github-info` / `--no-github-info`; on add, perform initial git inspect (as today) storing breakdown, and if enabled attempt GitHub fetch.
- `pjx set` / edit paths: accept flag to change `gh_enabled`; on save/edit of a project, refresh git+GitHub (when enabled) for that project so data is obtained as requested.
- `pjx list --format=delimited` / JSON export: include new fields (nullable) so consumers remain compatible (additive).

### Export/import & compatibility
- Export emits new nullable git/GitHub fields; `pjx import-file` accepts them (and tolerates absence). Update `docs/schemas/project-export.schema.json` accordingly.
- Existing DBs migrate forward automatically; no data loss (new columns default/null).

### Testing
- `internal/gitinfo`: unit tests for complete porcelain breakdown mapping.
- `internal/db`: migration v3, scan/add/update round-trip of new nullable fields.
- `internal/tui`: view/preview rendering includes mtime/git breakdown/PR-issue lines as applicable; key `R` triggers status refresh command.
- `internal/ghinfo` (or equivalent): parsing/detection logic unit-tested with fixtures/mocked command runner where practical; fetch failures are non-fatal.
- `make quality` / existing tests must pass.

## What’s Left to the Builder

- Exact compact formatting of the git breakdown line and where PR/Issues/Modified lines wrap in narrow vs wide preview (keep readable).
- Concrete `gh` command/flags and timeout values, and exact remote parsing edge cases (prefer `origin`, fall back reasonably).
- Whether `R` runs refresh synchronously with a status indicator or via an async command with progress message (must keep UI responsive).
- Final CLI flag naming consistent with existing style (`--github-info`/`--no-github-info` or `--gh/--no-gh`) and TUI edit-form control (cycle/toggle) for the tri-state.
- Any small list/delimited column additions kept additive and documented.

## Approval

Approved by user (“okey implementasi”, 2026-11-18) with breakpoint handling as specified above (narrow: list-first with compact git summary; full preview in two-column). Next: write implementation plan, then implement.