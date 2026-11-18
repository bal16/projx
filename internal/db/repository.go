package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/bal16/projx/internal/gitinfo"
	"github.com/bal16/projx/internal/model"
)

const projectColumns = `id,name,path,editor,profile,tags,group_name,favorite,last_opened,git_branch,git_staged,git_modified,git_untracked,git_deleted,git_renamed,git_conflicted,git_dirty,gh_enabled,gh_open_prs,gh_open_issues,gh_fetched_at,remote,created_at,updated_at`

// Repository provides CRUD operations for projects and settings.
type Repository struct {
	db *DB
}

// NewRepository creates a repository bound to db.
func NewRepository(db *DB) *Repository {
	return &Repository{db: db}
}

// --- Projects ---

// List returns all projects, optionally filtered by tag or group.
func (r *Repository) List(tag, group string, favoritesOnly bool) ([]model.Project, error) {
	q := `SELECT ` + projectColumns + ` FROM projects`
	var args []any
	var conds []string
	if group != "" {
		conds = append(conds, "group_name = ?")
		args = append(args, group)
	}
	if favoritesOnly {
		conds = append(conds, "favorite = 1")
	}
	if len(conds) > 0 {
		q += " WHERE " + joinAnd(conds)
	}
	q += " ORDER BY favorite DESC, last_opened DESC, name ASC"

	rows, err := r.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var out []model.Project
	for rows.Next() {
		p, err := scanProject(rows)
		if err != nil {
			return nil, err
		}
		if tag != "" && !p.HasTag(tag) {
			continue
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// Get returns a project by id, name (case-insensitive), or path.
// If multiple projects share the same name, an error is returned.
func (r *Repository) Get(idOrName string) (*model.Project, error) {
	// Prefer exact id match.
	row := r.db.QueryRow(
		`SELECT `+projectColumns+`
		 FROM projects WHERE id = ? LIMIT 1`, idOrName)
	p, err := scanProject(row)
	if err == nil {
		return &p, nil
	}
	// Fall back to name match, but detect ambiguity.
	rows, err := r.db.Query(
		`SELECT `+projectColumns+`
		 FROM projects WHERE LOWER(name) = LOWER(?)`, idOrName)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var matches []model.Project
	for rows.Next() {
		pp, err := scanProject(rows)
		if err != nil {
			return nil, err
		}
		matches = append(matches, pp)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(matches) == 0 {
		// Fall back to path match: callers (launcher, scripts) often pass the
		// project path as the identifier.
		if byPath, perr := r.GetByPath(idOrName); perr == nil {
			return byPath, nil
		}
		return nil, fmt.Errorf("project not found: %s", idOrName)
	}
	if len(matches) > 1 {
		names := make([]string, 0, len(matches))
		for _, m := range matches {
			names = append(names, m.ID)
		}
		return nil, fmt.Errorf("ambiguous project name %q; use id: %s", idOrName, strings.Join(names, ", "))
	}
	return &matches[0], nil
}

// GetByPath returns a project by its path.
func (r *Repository) GetByPath(path string) (*model.Project, error) {
	row := r.db.QueryRow(
		`SELECT `+projectColumns+`
		 FROM projects WHERE path = ? LIMIT 1`, path)
	p, err := scanProject(row)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// Add inserts a new project. Returns error if path already exists.
func (r *Repository) Add(p *model.Project) error {
	if p.ID == "" {
		p.ID = newID()
	}
	now := time.Now().Unix()
	if p.CreatedAt.IsZero() {
		p.CreatedAt = time.Unix(now, 0)
	}
	p.UpdatedAt = time.Unix(now, 0)

	tagsJSON, _ := json.Marshal(p.Tags)
	if p.Tags == nil {
		tagsJSON = []byte("[]")
	}

	_, err := r.db.Exec(
		`INSERT INTO projects (id,name,path,editor,profile,tags,group_name,favorite,last_opened,git_branch,git_staged,git_modified,git_untracked,git_deleted,git_renamed,git_conflicted,git_dirty,gh_enabled,gh_open_prs,gh_open_issues,gh_fetched_at,remote,created_at,updated_at)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		p.ID, p.Name, p.Path, p.Editor, p.Profile, string(tagsJSON), p.Group,
		boolToInt(p.Favorite), timeToUnix(p.LastOpened), p.GitBranch,
		p.GitStaged, p.GitModified, p.GitUntracked, p.GitDeleted, p.GitRenamed, p.GitConflicted, p.GitDirty,
		boolPtrToNullInt(p.GHEnabled), intPtrToNullInt(p.GHOpenPRs), intPtrToNullInt(p.GHOpenIssues), timePtrToNullUnix(p.GHFetchedAt),
		p.Remote, p.CreatedAt.Unix(), p.UpdatedAt.Unix(),
	)
	return err
}

// Update modifies an existing project.
func (r *Repository) Update(p *model.Project) error {
	p.UpdatedAt = time.Now()
	tagsJSON, _ := json.Marshal(p.Tags)
	if p.Tags == nil {
		tagsJSON = []byte("[]")
	}
	_, err := r.db.Exec(
		`UPDATE projects SET name=?,path=?,editor=?,profile=?,tags=?,group_name=?,favorite=?,last_opened=?,git_branch=?,git_staged=?,git_modified=?,git_untracked=?,git_deleted=?,git_renamed=?,git_conflicted=?,git_dirty=?,gh_enabled=?,gh_open_prs=?,gh_open_issues=?,gh_fetched_at=?,remote=?,updated_at=? WHERE id=?`,
		p.Name, p.Path, p.Editor, p.Profile, string(tagsJSON), p.Group,
		boolToInt(p.Favorite), timeToUnix(p.LastOpened), p.GitBranch,
		p.GitStaged, p.GitModified, p.GitUntracked, p.GitDeleted, p.GitRenamed, p.GitConflicted, p.GitDirty,
		boolPtrToNullInt(p.GHEnabled), intPtrToNullInt(p.GHOpenPRs), intPtrToNullInt(p.GHOpenIssues), timePtrToNullUnix(p.GHFetchedAt),
		p.Remote, p.UpdatedAt.Unix(), p.ID,
	)
	return err
}

// Remove deletes a project by id or name.
func (r *Repository) Remove(idOrName string) error {
	res, err := r.db.Exec("DELETE FROM projects WHERE id = ? OR LOWER(name) = LOWER(?)", idOrName, idOrName)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("project not found: %s", idOrName)
	}
	return nil
}

// TouchLastOpened updates last_opened timestamp.
func (r *Repository) TouchLastOpened(id string) error {
	_, err := r.db.Exec("UPDATE projects SET last_opened = ? WHERE id = ?", time.Now().Unix(), id)
	return err
}

// UpdateGitStatus caches git status breakdown for a project.
func (r *Repository) UpdateGitStatus(id string, gi gitinfo.Info) error {
	_, err := r.db.Exec(`UPDATE projects SET git_branch=?, git_staged=?, git_modified=?, git_untracked=?, git_deleted=?, git_renamed=?, git_conflicted=?, git_dirty=?, updated_at=? WHERE id=?`,
		gi.Branch, gi.Staged, gi.Modified, gi.Untracked, gi.Deleted, gi.Renamed, gi.Conflicted, gi.Dirty, time.Now().Unix(), id)
	return err
}

// Settings
func (r *Repository) GetSetting(key string) (string, error) {
	var v string
	err := r.db.QueryRow("SELECT value FROM settings WHERE key = ?", key).Scan(&v)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return v, err
}

func (r *Repository) SetSetting(key, value string) error {
	_, err := r.db.Exec("INSERT INTO settings (key,value) VALUES (?,?) ON CONFLICT(key) DO UPDATE SET value=excluded.value", key, value)
	return err
}

// --- helpers ---

// scanner abstracts *sql.Rows and *sql.Row for shared scanning.
type scanner interface {
	Scan(dest ...any) error
}

func scanProject(s scanner) (model.Project, error) {
	var p model.Project
	var tagsJSON string
	var fav, lastOpened, created, updated int64
	var gitStaged, gitModified, gitUntracked, gitDeleted, gitRenamed, gitConflicted, gitDirty int
	var ghEnabled, ghOpenPRs, ghOpenIssues, ghFetchedAt sql.NullInt64
	err := s.Scan(&p.ID, &p.Name, &p.Path, &p.Editor, &p.Profile, &tagsJSON, &p.Group,
		&fav, &lastOpened, &p.GitBranch,
		&gitStaged, &gitModified, &gitUntracked, &gitDeleted, &gitRenamed, &gitConflicted, &gitDirty,
		&ghEnabled, &ghOpenPRs, &ghOpenIssues, &ghFetchedAt,
		&p.Remote, &created, &updated)
	if err != nil {
		return p, err
	}
	p.Favorite = fav == 1
	p.GitStaged = gitStaged
	p.GitModified = gitModified
	p.GitUntracked = gitUntracked
	p.GitDeleted = gitDeleted
	p.GitRenamed = gitRenamed
	p.GitConflicted = gitConflicted
	p.GitDirty = gitDirty
	p.GHEnabled = nullIntToBoolPtr(ghEnabled)
	p.GHOpenPRs = nullIntToIntPtr(ghOpenPRs)
	p.GHOpenIssues = nullIntToIntPtr(ghOpenIssues)
	p.GHFetchedAt = nullIntToTimePtr(ghFetchedAt)
	p.LastOpened = unixToTime(lastOpened)
	p.CreatedAt = unixToTime(created)
	p.UpdatedAt = unixToTime(updated)
	_ = json.Unmarshal([]byte(tagsJSON), &p.Tags)
	if p.Tags == nil {
		p.Tags = []string{}
	}
	return p, nil
}

func joinAnd(conds []string) string {
	out := conds[0]
	for i := 1; i < len(conds); i++ {
		out += " AND " + conds[i]
	}
	return out
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func boolPtrToNullInt(b *bool) any {
	if b == nil {
		return nil
	}
	if *b {
		return 1
	}
	return 0
}

func intPtrToNullInt(v *int) any {
	if v == nil {
		return nil
	}
	return *v
}

func timePtrToNullUnix(t *time.Time) any {
	if t == nil || t.IsZero() {
		return nil
	}
	return t.Unix()
}

func nullIntToBoolPtr(v sql.NullInt64) *bool {
	if !v.Valid {
		return nil
	}
	b := v.Int64 != 0
	return &b
}

func nullIntToIntPtr(v sql.NullInt64) *int {
	if !v.Valid {
		return nil
	}
	i := int(v.Int64)
	return &i
}

func nullIntToTimePtr(v sql.NullInt64) *time.Time {
	if !v.Valid || v.Int64 == 0 {
		return nil
	}
	t := time.Unix(v.Int64, 0)
	return &t
}

func timeToUnix(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}
	return t.Unix()
}

func unixToTime(u int64) time.Time {
	if u == 0 {
		return time.Time{}
	}
	return time.Unix(u, 0)
}

func newID() string {
	return "p" + uuid.NewString()
}

// --- Sets (workspace sets) ---

// Set represents a named group of project paths opened together.
type Set struct {
	Name  string   `json:"name"`
	Paths []string `json:"paths"`
}

// ListSets returns all sets.
func (r *Repository) ListSets() ([]Set, error) {
	rows, err := r.db.Query("SELECT name, paths FROM sets ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []Set
	for rows.Next() {
		var s Set
		var pathsJSON string
		if err := rows.Scan(&s.Name, &pathsJSON); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(pathsJSON), &s.Paths)
		if s.Paths == nil {
			s.Paths = []string{}
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// GetSet returns a set by name.
func (r *Repository) GetSet(name string) (*Set, error) {
	row := r.db.QueryRow("SELECT name, paths FROM sets WHERE name = ?", name)
	var s Set
	var pathsJSON string
	if err := row.Scan(&s.Name, &pathsJSON); err != nil {
		return nil, err
	}
	_ = json.Unmarshal([]byte(pathsJSON), &s.Paths)
	if s.Paths == nil {
		s.Paths = []string{}
	}
	return &s, nil
}

// AddSet creates or updates a set.
func (r *Repository) AddSet(name string, paths []string) error {
	now := time.Now().Unix()
	b, _ := json.Marshal(paths)
	_, err := r.db.Exec(
		`INSERT INTO sets (name, paths, created_at, updated_at) VALUES (?,?,?,?)
		 ON CONFLICT(name) DO UPDATE SET paths=excluded.paths, updated_at=excluded.updated_at`,
		name, string(b), now, now,
	)
	return err
}

// RemoveSet deletes a set.
func (r *Repository) RemoveSet(name string) error {
	res, err := r.db.Exec("DELETE FROM sets WHERE name = ?", name)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("set not found: %s", name)
	}
	return nil
}
