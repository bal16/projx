package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/bal16/projx/internal/ghinfo"
	"github.com/bal16/projx/internal/gitinfo"
	"github.com/bal16/projx/internal/model"
)

type addParams struct {
	name       string
	editor     string
	profile    string
	group      string
	tags       []string
	favorite   bool
	ghEnabled  *bool
}

func runAdd(a *App, path string, ap addParams) error {
	if strings.HasPrefix(path, "~/") {
		home, _ := os.UserHomeDir()
		path = filepath.Join(home, path[2:])
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	if info, err := os.Stat(abs); err != nil || !info.IsDir() {
		return fmt.Errorf("path is not a directory: %s", abs)
	}
	if existing, _ := a.Repo.GetByPath(abs); existing != nil {
		return fmt.Errorf("project already exists: %s (%s)", existing.Name, existing.ID)
	}
	name := ap.name
	if name == "" {
		name = filepath.Base(abs)
	}
	p := &model.Project{
		Name:      name,
		Path:      abs,
		Editor:    ap.editor, // per-project override only; leave empty to follow default_editor
		Profile:   ap.profile,
		Tags:      ap.tags,
		Group:     ap.group,
		Favorite:  ap.favorite,
		GHEnabled: ap.ghEnabled,
	}
	if ap.tags == nil {
		p.Tags = []string{}
	}
	if gi := gitinfo.Inspect(abs); gi.IsRepo {
		p.GitBranch = gi.Branch
		p.GitStaged = gi.Staged
		p.GitModified = gi.Modified
		p.GitUntracked = gi.Untracked
		p.GitDeleted = gi.Deleted
		p.GitRenamed = gi.Renamed
		p.GitConflicted = gi.Conflicted
		p.GitDirty = gi.Dirty
	}
	if err := a.Repo.Add(p); err != nil {
		return err
	}
	if p.GHEnabled != nil && *p.GHEnabled {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		if c, err := ghinfo.FetchOpenCounts(ctx, p.Path); err == nil {
			_ = a.Repo.UpdateGitHubStatus(p.ID, c.OpenPRs, c.OpenIssues, c.FetchedAt)
		}
		cancel()
	}
	fmt.Printf("added project %q (%s)\n", p.Name, p.ID)
	return nil
}

func newAddCmd() *cobra.Command {
	var ap addParams
	var ghInfo, noGHInfo bool
	cmd := &cobra.Command{
		Use:   "add <path>",
		Short: "Add a project",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if cmd.Flags().Changed("github-info") {
				v := true
				ap.ghEnabled = &v
			}
			if cmd.Flags().Changed("no-github-info") {
				v := false
				ap.ghEnabled = &v
			}
			return runAdd(requireApp(), args[0], ap)
		},
	}
	cmd.Flags().StringVar(&ap.name, "name", "", "display name / alias")
	cmd.Flags().StringVar(&ap.editor, "editor", "", "preferred editor (zed, code, nvim, ...)")
	cmd.Flags().StringVar(&ap.profile, "profile", "", "editor profile (VS Code profiles)")
	cmd.Flags().StringSliceVar(&ap.tags, "tag", nil, "tag (repeatable)")
	cmd.Flags().StringVar(&ap.group, "group", "", "group name (e.g. Work/Frontend)")
	cmd.Flags().BoolVar(&ap.favorite, "favorite", false, "mark as favorite")
	cmd.Flags().BoolVar(&ghInfo, "github-info", false, "fetch and cache open GitHub PR/issue counts (requires gh)")
	cmd.Flags().BoolVar(&noGHInfo, "no-github-info", false, "disable GitHub info fetch for this project")
	return cmd
}
