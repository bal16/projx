# TUI Project Info Improvements Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Enrich the TUI project preview with live root mtime, a complete cached git status breakdown, and optional per-project cached GitHub open PR/issue counts, plus a trigger to refresh status for all projects.

**Architecture:** Extend the existing SQLite-backed project model/cache (schema v3 migration, additive nullable columns) and broaden `gitinfo` to parse full `git status --porcelain` counts. Add a small `internal/ghinfo` helper that detects GitHub remotes and, when a per-project opt-in (`gh_enabled`) is set, fetches open PR/issue counts via the `gh` CLI with graceful failure. Surface data in the TUI preview/list (live mtime via `os.Stat`), wire refresh-on-add/edit/per-project and a new “refresh all status” action (`R`/`pjx refresh`), and extend export/list outputs compatibly.

**Tech Stack:** Go 1.26, SQLite (`modernc.org/sqlite`), Bubble Tea/Lip Gloss TUI, Cobra CLI, `os/exec` for `git`/`gh`, standard library (`os`, `time`, `strings`).

**Spec:** `docs/superpowers/specs/2026-11-18-tui-project-info-design.md` — the plan argues from the spec; executors read both.

## Global Constraints

- **Schema:** current `schemaVersion` is 2; add an idempotent v3 migration (additive `ALTER TABLE` / default/null-safe) and bump to 3. Do not drop or rewrite existing columns/data.
- **Compatibility:** keep `git_dirty` as the derived total (`staged+modified+untracked+deleted+renamed+conflicted`) for existing consumers; `pjx export`/`pjx import-file` and delimited output remain backward-compatible (new fields additive/nullable/omittable).
- **GitHub fetch:** optional **per project** (`gh_enabled` tri-state: `NULL` unset/off, `1` enabled, `0` disabled). Fetch counts only when enabled and remote resolves to GitHub; use `gh` CLI; on any failure (no `gh`, unauthenticated, network/parse error, non-GitHub) do **not** fail the parent operation and leave cached `gh_open_prs`/`gh_open_issues`/`gh_fetched_at` unchanged/null.
- **Mtime:** project “last modified” is `os.Stat(project.Path).ModTime()` of the project root, computed live for display (not stored).
- **TUI responsiveness:** status refresh (especially “all projects”) runs via a command/subprocess path (not a blocking UI update); `r` keeps “reload from DB”, `R` triggers status refresh for all.
- **Responsive:** two-column when width allows (list capped 90, preview gets remainder, values truncated to panel); narrow/single-column stays list-first with compact git summary in list rows; small height prioritizes key preview fields (Name/Path, Git, Modified, PR/Issues).
- **Quality:** `GOFLAGS=-mod=mod GOPROXY=off go test ./...` and `~/go/bin/golangci-lint run ./...` (or `make quality`) must pass.

## Review Focus

- **Porcelain mapping completeness:** a line with unmerged status (`UU`, `AA`, `DD`, any `U`) must count as conflicted (and not be miscounted as plain modified); `??` untracked; renames/copies counted as renamed/staged per spec — verify with fixtures, not just happy-path modified files.
- **Graceful GitHub failure:** `gh` missing, non-zero exit, or non-GitHub remote during add/edit/refresh must not error the command or TUI save; cached fields remain null/prior and other info (git/mtime) still displays.
- **Nullable/tri-state semantics:** `gh_enabled == NULL` means off (no fetch, PR/Issues row omitted); enabled-but-never-fetched shows placeholder/null (not fabricated zeros); import of older exports missing new fields must still work.
- **Signature/caller consistency:** `UpdateGitStatus` and `gitinfo.Info` change shape — every caller (refresh, list --refresh, add, scan, TUI/CLI flows) must compile and persist the full breakdown + derived `git_dirty` total.
- **Narrow/height truncation:** long breakdown/PR lines must truncate to available width without panics or layout breakage; key fields (Git, Modified, PR/Issues) appear before lower-priority lines so they survive small `maxLines`.

---

### Task 1: Model and DB schema v3 (breakdown + GitHub cache fields)

**Files:**
- Modify: `internal/model/project.go`
- Modify: `internal/db/migrations.go`
- Modify: `internal/db/db.go` (only if needed; DSN unchanged)
- Modify: `internal/db/repository.go`
- Test: `internal/db/repository_test.go`

**Interfaces:**
- Produces: `model.Project` fields: `GitStaged, GitModified, GitUntracked, GitDeleted, GitRenamed, GitConflicted int` (json `git_staged,omitempty` etc.), `GitDirty int` kept as total; `GHEnabled *bool` (`json:"gh_enabled,omitempty"`) for tri-state (`nil` unset/off, `&true` enabled, `&false` explicitly disabled); `GHOpenPRs *int` (`json:"gh_open_prs,omitempty"`), `GHOpenIssues *int` (`json:"gh_open_issues,omitempty"`); `GHFetchedAt *time.Time` (`json:"gh_fetched_at,omitempty"`, RFC3339 in JSON like other timestamps, stored as nullable unix int in DB). (Exact tag names must match schema/export additions in Task 7.)
- Produces: `(*Repository).UpdateGitStatus` will be replaced in Task 2; in this task keep DB able to store/round-trip new fields via `Add`/`Update`/scan and add a temporary-compatible helper if needed so repo compiles incrementally — prefer introducing `UpdateGitBreakdown` in Task 2 and adapting callers then. (Plan implementers: Task 1+2 may be executed as a coupled pair to keep `go build ./...` green; if so, treat their commits as `refactor(db,gitinfo): ...` + `feat(model): ...` in order, but do not leave the tree uncompilable.)
- Consumes: existing `Project`/repo API.

- [ ] **Step 1: Write failing DB round-trip/migration tests**

In `internal/db/repository_test.go`, add `TestGitBreakdownAndGHRoundTrip` that opens a temp DB (`tempDB`), inserts a project, sets breakdown + GitHub nullable fields via repo update/save path you introduce (e.g. populate fields on `model.Project` and `repo.Update`, and/or a new setter), then `repo.Get` and assert all new values (including `GHEnabled=nil` vs true/false, `GHOpenPRs/Issues/FetchedAt` null vs set) persist. Add `TestMigrateV2ToV3` (or equivalent) that simulates/relies on opening DB: after `Open` on a fresh DB, `schema_version` is 3 and new columns exist (e.g. query `PRAGMA table_info(projects)` or select the columns).

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /home/bal/workspace/project/project-manager-ext && GOFLAGS=-mod=mod GOPROXY=off go test ./internal/db -run 'TestGitBreakdownAndGHRoundTrip|TestMigrateV2ToV3' -v`
Expected: FAIL (missing fields/columns/methods).

- [ ] **Step 3: Implement model fields and schema v3 migration**

In `internal/model/project.go` add the new fields with JSON tags. In `internal/db/migrations.go` bump `schemaVersion` to 3 and in `migrate()` add idempotent v2→v3 `ALTER TABLE projects ADD COLUMN ...` for each new column (ignore duplicate-column errors as done for v2) before setting version. Ensure defaults/nullability match spec (`gh_*` nullable).

- [ ] **Step 4: Implement repository scan/add/update for new fields**

In `internal/db/repository.go` extend `SELECT`/`Scan` column lists, `scanProject`, `Add`, and `Update` to read/write the new columns. Handle nullable `gh_enabled`/counts/fetched_at (use `sql.NullInt64` / `sql.NullTime`/`NullInt64` unix as appropriate) mapping to pointer fields. Keep `git_dirty` stored as provided (caller will set total). Keep `List`/`Get`/`GetByPath` queries consistent.

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd /home/bal/workspace/project/project-manager-ext && GOFLAGS=-mod=mod GOPROXY=off go test ./internal/db -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/model/project.go internal/db/migrations.go internal/db/repository.go internal/db/repository_test.go
git commit -m "feat(db): add v3 schema for git breakdown and GitHub cache"
```

---

### Task 2: Git status breakdown parsing and repository/CLI wiring

**Files:**
- Modify: `internal/gitinfo/gitinfo.go`
- Modify: `internal/gitinfo/gitinfo_test.go`
- Modify: `internal/db/repository.go` (replace `UpdateGitStatus`)
- Modify: `internal/cli/refresh.go`
- Modify: `internal/cli/list.go`
- Modify: `internal/cli/add.go`
- Modify: `internal/cli/scan.go` (if it sets git fields; confirm and update)
- Modify: any other callers of `UpdateGitStatus`/`Inspect` git fields (search repo)

**Interfaces:**
- Produces: `gitinfo.Info{Branch string; Staged, Modified, Untracked, Deleted, Renamed, Conflicted, Dirty int; IsRepo bool}` with `Dirty`/total = sum of buckets (or `Total()` helper) matching spec mapping.
- Produces: `(*Repository).UpdateGitStatus(id string, gi gitinfo.Info) error` (or equivalent taking breakdown) persisting branch, all breakdown fields, and `git_dirty` total + `updated_at`.
- Consumes: Task 1 model/DB columns.

- [ ] **Step 1: Write failing gitinfo breakdown tests**

In `internal/gitinfo/gitinfo_test.go`, add table/individual tests covering: untracked `??` → Untracked; unstaged modify → Modified; staged add/modify → Staged; deleted (unstaged/staged as mapped) → Deleted; rename/copy → Renamed; unmerged/conflict codes (e.g. `UU`) → Conflicted; mixed porcelain yields correct totals. Use temp repos with `git init/config/add/commit` and file mutations (skip if `git` unavailable, consistent with existing tests).

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /home/bal/workspace/project/project-manager-ext && GOFLAGS=-mod=mod GOPROXY=off go test ./internal/gitinfo -v`
Expected: FAIL (missing fields/counts).

- [ ] **Step 3: Implement porcelain parsing and Info fields**

In `internal/gitinfo/gitinfo.go` extend `Info` and `Inspect` to classify each `--porcelain` line per spec (conflicted first, then untracked, then renamed/copied, staged index changes, unstaged modified/deleted) and compute `Dirty` as total. Preserve existing `Branch`/`IsRepo` behavior.

- [ ] **Step 4: Run gitinfo tests**

Run: `cd /home/bal/workspace/project/project-manager-ext && GOFLAGS=-mod=mod GOPROXY=off go test ./internal/gitinfo -v`
Expected: PASS.

- [ ] **Step 5: Replace repo UpdateGitStatus and update all callers**

Change `UpdateGitStatus` in `internal/db/repository.go` to accept the breakdown (e.g. `gitinfo.Info` or explicit counts) and persist all git columns + total. Update every caller (`internal/cli/refresh.go`, `internal/cli/list.go` refresh path, `internal/cli/add.go`, `internal/cli/scan.go`, and any others found by `rg UpdateGitStatus|GitDirty`) to pass the new info and, where they cache onto in-memory `model.Project`, set all breakdown fields + total. Adjust refresh/list text output if needed to remain sensible (may still show total or a compact breakdown; keep non-breaking).

- [ ] **Step 6: Run build and tests**

Run: `cd /home/bal/workspace/project/project-manager-ext && GOFLAGS=-mod=mod GOPROXY=off go test ./internal/db ./internal/gitinfo ./internal/cli ./internal/...`
Expected: PASS (fix any compile issues).

- [ ] **Step 7: Commit**

```bash
git add internal/gitinfo internal/db/repository.go internal/cli/refresh.go internal/cli/list.go internal/cli/add.go internal/cli/scan.go
git commit -m "feat(gitinfo): parse full status breakdown and persist it"
```

(Include any other updated caller files in the same commit.)

---

### Task 3: GitHub info helper (detect repo + fetch counts, graceful)

**Files:**
- Create: `internal/ghinfo/ghinfo.go`
- Create: `internal/ghinfo/ghinfo_test.go`

**Interfaces:**
- Produces: `type Counts struct { OpenPRs, OpenIssues int; FetchedAt time.Time }` (or pointer-returning form) and functions such as `DetectRepo(path string) (ownerRepo string, ok bool)` (parses `origin`/remote URL for `github.com`) and `FetchOpenCounts(ctx context.Context, path string) (Counts, error)` / `FetchIfEnabled(...)` that runs `gh pr list -R <repo> --state open --json number --jq 'length'` and `gh issue list -R <repo> --state open --json number --jq 'length'` (or `--limit`/jq equivalent) with a timeout; returns error on failure so callers can choose to ignore. Helper must be safe when `gh` missing/non-GitHub.
- Consumes: project path; no DB dependency.

- [ ] **Step 1: Write failing detection tests**

In `internal/ghinfo/ghinfo_test.go` add tests for parsing GitHub remote URLs: `https://github.com/owner/repo.git`, `git@github.com:owner/repo.git`, `ssh://git@github.com/owner/repo`, and non-GitHub/empty returning not-ok. (Test pure parsing/detection helpers; do not require network.)

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /home/bal/workspace/project/project-manager-ext && GOFLAGS=-mod=mod GOPROXY=off go test ./internal/ghinfo -v`
Expected: FAIL (package/functions missing).

- [ ] **Step 3: Implement detection and fetch helpers**

Implement URL parsing + remote resolution (`git -C <path> remote get-url origin`, fallback first remote if desired). Implement fetch using `exec.CommandContext` with timeout (e.g. 5–10s) invoking `gh ... -R owner/repo`; parse integer output. On any error return it (callers will treat as graceful skip). Keep functions small/testable.

- [ ] **Step 4: Run tests**

Run: `cd /home/bal/workspace/project/project-manager-ext && GOFLAGS=-mod=mod GOPROXY=off go test ./internal/ghinfo -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/ghinfo
git commit -m "feat(ghinfo): detect GitHub repos and fetch open PR/issue counts"
```

---

### Task 4: Wire GitHub cache into refresh/add/set and repository helpers

**Files:**
- Modify: `internal/db/repository.go` (add method to persist GitHub counts/fetched_at, e.g. `UpdateGitHubStatus(id string, prs, issues int, fetchedAt time.Time) error` and/or nullable-aware setter)
- Modify: `internal/cli/refresh.go`
- Modify: `internal/cli/add.go`
- Modify: `internal/cli/set.go`
- Modify: `internal/cli/scan.go` (if add/inspect path needs gh behavior; likely only on explicit add paths — follow spec)
- Test: `internal/db/repository_test.go` (if new repo method), CLI tests if present/feasible

**Interfaces:**
- Consumes: `ghinfo.DetectRepo`/fetch, `model.Project.GHEnabled` and nullable cache fields.
- Produces: behavior per spec — `pjx refresh [project...]` refreshes git for targets and, when `p.GHEnabled != nil && *p.GHEnabled == true`, attempts GitHub fetch and persists counts+fetched_at on success (ignore failure). `pjx add` gains `--github-info` / `--no-github-info` setting `GHEnabled` (null when omitted) and, after add, performs git inspect + GitHub fetch when enabled. `pjx set` gains flag to change `GHEnabled` and on save/edit refreshes git (+ GitHub if enabled) for that project so data is obtained.

- [ ] **Step 1: Add repo method to persist GitHub counts**

Implement nullable-safe update (e.g. set `gh_open_prs`, `gh_open_issues`, `gh_fetched_at`) and, if needed, a method to set `gh_enabled`. Cover with a small DB test asserting values persist and can be nulled where appropriate.

- [ ] **Step 2: Run DB tests**

Run: `cd /home/bal/workspace/project/project-manager-ext && GOFLAGS=-mod=mod GOPROXY=off go test ./internal/db -v`
Expected: PASS.

- [ ] **Step 3: Extend CLI refresh/add/set**

- `refresh`: after updating git status for each target, if enabled attempt `ghinfo` fetch and save on success; keep per-project skip-on-error behavior.
- `add`: add flags, set `GHEnabled`, after `repo.Add` run git inspect + `UpdateGitStatus` as today and attempt GitHub fetch when enabled (persist if success).
- `set`: add flags to set/clear/enable `GHEnabled`; after `repo.Update` re-inspect git and (if enabled) fetch GitHub for the project, persisting results.
- Ensure flag naming matches Cobra style and is documented in help (final name consistent: e.g. `--github-info`/`--no-github-info`).

- [ ] **Step 4: Run CLI and full tests/build**

Run: `cd /home/bal/workspace/project/project-manager-ext && GOFLAGS=-mod=mod GOPROXY=off go build ./... && GOFLAGS=-mod=mod GOPROXY=off go test ./internal/cli ./internal/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/db/repository.go internal/db/repository_test.go internal/cli/add.go internal/cli/set.go internal/cli/refresh.go internal/cli/scan.go
git commit -m "feat(cli): cache GitHub info on add, edit, and refresh"
```

(Include only files actually changed.)

---

### Task 5: TUI preview and list — mtime, git breakdown, PR/Issues

**Files:**
- Modify: `internal/tui/view.go`
- Modify: `internal/tui/styles.go` (only if needed)
- Test: `internal/tui/tui_test.go`

**Interfaces:**
- Consumes: `model.Project` new fields; live `os.Stat(p.Path).ModTime()` for Modified (fallback `-` on error).
- Produces: preview shows `Modified`, `Git: <branch> ...` with complete compact breakdown (prefer non-zero buckets, include all categories per spec formatting), and `PR`/`Issues` line per `gh_enabled`/null rules. List rows show compact git summary (branch + non-zero counts) instead of/alongside total `+dirty` while remaining readable and truncated. Respects two-column/narrow/small-height behavior (key fields first, truncation to panel width).

- [ ] **Step 1: Write/update failing TUI view tests**

Add tests in `internal/tui/tui_test.go` asserting: a project with breakdown/GitHub data renders `Modified:` (formatted), `Git:` line containing branch and expected bucket tokens (e.g. `M:`, `?:`, `S:`), and `PR:`/`Issues:` when enabled+set; a project with `gh_enabled` true but null counts shows placeholder/null form (or omits per spec) — pick one consistent behavior and assert it; a disabled/null-enabled project omits PR/Issues. Assert list row includes compact git summary.

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /home/bal/workspace/project/project-manager-ext && GOFLAGS=-mod=mod GOPROXY=off go test ./internal/tui -v`
Expected: FAIL.

- [ ] **Step 3: Implement view rendering helpers and layout**

Add helpers (e.g. `formatGitBreakdown(p model.Project) string`, `formatModified(path string) string`, `formatGH(p model.Project) (string, bool)`) and update `View()` preview construction (insert Modified/Git/PR-Issues in prioritized order) and list row git segment. Ensure truncation uses available widths (`truncate`/`padRight`) and narrow mode remains list-first with compact summary.

- [ ] **Step 4: Run TUI tests**

Run: `cd /home/bal/workspace/project/project-manager-ext && GOFLAGS=-mod=mod GOPROXY=off go test ./internal/tui -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/tui/view.go internal/tui/tui_test.go internal/tui/styles.go
git commit -m "feat(tui): show mtime, git breakdown, and GitHub info in preview"
```

---

### Task 6: TUI edit GitHub toggle and refresh-all status (`R`)

**Files:**
- Modify: `internal/tui/edit.go`
- Modify: `internal/tui/update.go`
- Modify: `internal/tui/commands.go`
- Modify: `internal/tui/model.go` (if status/fields needed)
- Modify: `internal/tui/view.go` (footer/help hints)
- Test: `internal/tui/tui_test.go`

**Interfaces:**
- Produces: edit form includes a control for per-project GitHub fetch (tri-state/null-able: unset/off, on, off-explicit — UI may cycle values; persisted to `GHEnabled`). On save, project is updated and git (+ GitHub if enabled) is refreshed for that project (via repo+gitinfo+ghinfo call or by invoking CLI `set`/refresh logic consistently) so data is obtained as requested, then list reloads.
- Produces: `R` in normal mode triggers **refresh status for all projects** (git + GitHub where enabled) asynchronously (e.g. `m.runCommand("refresh")` or a dedicated tea.Cmd that runs refresh and emits result) and reloads; `r` remains reload-from-DB. Help line and `?` output include `R: refresh status`.

- [ ] **Step 1: Write failing tests for edit toggle and R behavior**

Add tests: edit form renders/exposes GitHub option and saving with it enabled/disabled updates `GHEnabled`; pressing `R` in normal mode returns a command that performs refresh (assert via command/result handling pattern used in existing tests or by invoking the command-producing path) and status/help mention `R`.

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /home/bal/workspace/project/project-manager-ext && GOFLAGS=-mod=mod GOPROXY=off go test ./internal/tui -v`
Expected: FAIL.

- [ ] **Step 3: Implement edit form field and save refresh**

Extend `EditForm` (inputs/cycle or text field) to edit `GHEnabled` (accept values like empty/unset, on/true, off/false; display clearly). On save, set field on project, persist via repo, then refresh git and conditionally GitHub for that project (reusing `gitinfo`/`ghinfo` helpers or CLI-equivalent logic) before reloading list; keep failure of GitHub fetch non-fatal.

- [ ] **Step 4: Implement R key, help, and hints**

In `handleNormalMode` add `case "R":` to set status (e.g. "refreshing status...") and return `m.runCommand("refresh")` (or equivalent async refresh command) so it refreshes all and reloads on completion. Update footer hint strings in `View` and the `?` help text to document `r` (reload) vs `R` (refresh status). Optionally add `refresh` to command palette suggestions if missing (it exists in base list).

- [ ] **Step 5: Run TUI tests**

Run: `cd /home/bal/workspace/project/project-manager-ext && GOFLAGS=-mod=mod GOPROXY=off go test ./internal/tui -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/tui/edit.go internal/tui/update.go internal/tui/view.go internal/tui/commands.go internal/tui/model.go internal/tui/tui_test.go
git commit -m "feat(tui): add GitHub toggle and refresh-all status action"
```

---

### Task 7: CLI output, export/import schema, and docs

**Files:**
- Modify: `internal/cli/list.go` (text/delimited/json include new fields additively)
- Modify: `internal/cli/export.go` (no logic change expected, but verify round-trip)
- Modify: `docs/schemas/project-export.schema.json`
- Modify: `docs/USAGE.md` (document `R`/refresh behavior, `--github-info` flags if appropriate)
- Modify: `docs/CONFIGURATION.md` or README as needed (brief mention) — only if consistent with existing docs style

**Interfaces:**
- Produces: delimited format remains additive/backward-friendly (append new fields at end or document stable order; do not break existing field positions 0..9 if consumers depend on them — safest to append new columns after existing ones). JSON export/import carries new nullable fields. Schema documents new properties: `git_staged`, `git_modified`, `git_untracked`, `git_deleted`, `git_renamed`, `git_conflicted`, `gh_enabled`, `gh_open_prs`, `gh_open_issues`, `gh_fetched_at`.

- [ ] **Step 1: Update schema and (if needed) tests/expectations**

Add new properties to `docs/schemas/project-export.schema.json` with correct types/nullability and descriptions. Update any snapshot/golden expectations if present.

- [ ] **Step 2: Adjust list/delimited output**

In `internal/cli/list.go`, keep existing delimited fields in the same order and append new fields (breakdown, gh fields, fetched_at) at the end so existing parsers reading fixed early indices still work. Update human-readable text to show useful breakdown/gh info when present (concise).

- [ ] **Step 3: Verify export/import round-trip**

Manually or via test: `pjx export` includes new fields when set; `pjx import-file` of an export without new fields still imports (fields null/default). Run: `GOFLAGS=-mod=mod GOPROXY=off go test ./internal/cli ./...`

- [ ] **Step 4: Update docs**

Document `R` (refresh status all) in TUI keys, `pjx refresh` refreshing GitHub when enabled, and `--github-info/--no-github-info` on `add`/`set` in USAGE/CLI docs as appropriate.

- [ ] **Step 5: Commit**

```bash
git add internal/cli/list.go internal/cli/export.go docs/schemas/project-export.schema.json docs/USAGE.md docs/CONFIGURATION.md README.md
git commit -m "feat(cli): expose status breakdown in output and update schema/docs"
```

(Include only files actually changed.)

---

### Task 8: Final verification and quality gate

**Files:**
- Modify: any files required to fix issues found (none expected)

**Interfaces:**
- Consumes: completed implementation.
- Produces: green build/tests/lint.

- [ ] **Step 1: Run formatters and linters**

Run: `cd /home/bal/workspace/project/project-manager-ext && gofmt -w . && ~/go/bin/golangci-lint run ./...`
Expected: clean.

- [ ] **Step 2: Run full test suite**

Run: `cd /home/bal/workspace/project/project-manager-ext && GOFLAGS=-mod=mod GOPROXY=off go test ./...`
Expected: PASS.

- [ ] **Step 3: Smoke-check TUI/CLI behavior**

Run quick manual checks if feasible: `GOFLAGS=-mod=mod GOPROXY=off go build ./cmd/pjx`, then `./bin/pjx refresh` (or on a known project), `./bin/pjx list`, and `./bin/pjx tui` to confirm preview shows Modified/Git breakdown and `R` works (GitHub lines appear only when enabled and fetchable).

- [ ] **Step 4: Commit any final fixups**

```bash
git add -A
git commit -m "chore: finalize TUI project info improvements"
```

(Only if there are fixup changes; otherwise skip.)