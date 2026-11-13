package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bal16/projx/internal/gitinfo"
	"github.com/bal16/projx/internal/model"
	"github.com/spf13/cobra"
)

func newAddCmd() *cobra.Command {
	var name, editor, profile, group string
	var tags []string
	var favorite bool

	cmd := &cobra.Command{
		Use:   "add <path>",
		Short: "Add a project",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a := requireApp()
			path := args[0]
			// expand ~
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

			// check duplicate
			if existing, _ := a.Repo.GetByPath(abs); existing != nil {
				return fmt.Errorf("project already exists: %s (%s)", existing.Name, existing.ID)
			}

			if name == "" {
				name = filepath.Base(abs)
			}
			if editor == "" {
				editor = a.Config.DefaultEditor
			}

			p := &model.Project{
				Name:     name,
				Path:     abs,
				Editor:   editor,
				Profile:  profile,
				Tags:     tags,
				Group:    group,
				Favorite: favorite,
			}
			// cache git info
			if gi := gitinfo.Inspect(abs); gi.IsRepo {
				p.GitBranch = gi.Branch
				p.GitDirty = gi.Dirty
			}

			if err := a.Repo.Add(p); err != nil {
				return err
			}
			fmt.Printf("added project %q (%s)\n", p.Name, p.ID)
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "display name / alias")
	cmd.Flags().StringVar(&editor, "editor", "", "preferred editor (zed, code, nvim, ...)")
	cmd.Flags().StringVar(&profile, "profile", "", "editor profile (VS Code profiles)")
	cmd.Flags().StringSliceVar(&tags, "tag", nil, "tag (repeatable)")
	cmd.Flags().StringVar(&group, "group", "", "group name (e.g. Work/Frontend)")
	cmd.Flags().BoolVar(&favorite, "favorite", false, "mark as favorite")
	return cmd
}
