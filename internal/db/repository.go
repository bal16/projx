package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/bal16/projx/internal/model"
)

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
	q := `SELECT id,name,path,editor,profile,tags,group_name,favorite,last_opened,git_branch,git_dirty,remote,created_at,updated_at FROM projects`
	var args []any
	var conds []string
	if tag != "" {
		conds = append(conds, "tags LIKE ?")
		args = append(args, "%\""+tag+"\"%")
	}
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
		out = append(out, p)
	}
	return out, rows.Err()
}

// Get returns a project by id or name (case-insensitive name match).
func (r *Repository) Get(idOrName string) (*model.Project, error) {
	row := r.db.QueryRow(
		`SELECT id,name,path,editor,profile,tags,group_name,favorite,last_opened,git_branch,git_dirty,remote,created_at,updated_at
		 FROM projects WHERE id = ? OR LOWER(name) = LOWER(?) LIMIT 1`, idOrName, idOrName)
	p, err := scanProject(row)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// GetByPath returns a project by its path.
func (r *Repository) GetByPath(path string) (*model.Project, error) {
	row := r.db.QueryRow(
		`SELECT id,name,path,editor,profile,tags,group_name,favorite,last_opened,git_branch,git_dirty,remote,created_at,updated_at
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
		`INSERT INTO projects (id,name,path,editor,profile,tags,group_name,favorite,last_opened,git_branch,git_dirty,remote,created_at,updated_at)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		p.ID, p.Name, p.Path, p.Editor, p.Profile, string(tagsJSON), p.Group,
		boolToInt(p.Favorite), timeToUnix(p.LastOpened), p.GitBranch, p.GitDirty, p.Remote,
		p.CreatedAt.Unix(), p.UpdatedAt.Unix(),
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
		`UPDATE projects SET name=?,path=?,editor=?,profile=?,tags=?,group_name=?,favorite=?,last_opened=?,git_branch=?,git_dirty=?,remote=?,updated_at=? WHERE id=?`,
		p.Name, p.Path, p.Editor, p.Profile, string(tagsJSON), p.Group,
		boolToInt(p.Favorite), timeToUnix(p.LastOpened), p.GitBranch, p.GitDirty, p.Remote,
		p.UpdatedAt.Unix(), p.ID,
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

// UpdateGitStatus caches git branch and dirty count.
func (r *Repository) UpdateGitStatus(id, branch string, dirty int) error {
	_, err := r.db.Exec("UPDATE projects SET git_branch=?, git_dirty=?, updated_at=? WHERE id=?",
		branch, dirty, time.Now().Unix(), id)
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
	var gitDirty int
	err := s.Scan(&p.ID, &p.Name, &p.Path, &p.Editor, &p.Profile, &tagsJSON, &p.Group,
		&fav, &lastOpened, &p.GitBranch, &gitDirty, &p.Remote, &created, &updated)
	if err != nil {
		return p, err
	}
	p.Favorite = fav == 1
	p.GitDirty = gitDirty
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
	// Simple unique id: timestamp + random-ish suffix.
	return fmt.Sprintf("p%d", time.Now().UnixNano())
}
