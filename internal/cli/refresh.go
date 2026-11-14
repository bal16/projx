package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/bal16/projx/internal/gitinfo"
)

func newRefreshCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "refresh [project]",
		Short: "Refresh cached git status for projects",
		RunE: func(cmd *cobra.Command, args []string) error {
			a := requireApp()
			var list []string
			if len(args) > 0 {
				list = args
			} else {
				projs, err := a.Repo.List("", "", false)
				if err != nil {
					return err
				}
				for _, p := range projs {
					list = append(list, p.ID)
				}
			}
			for _, id := range list {
				p, err := a.Repo.Get(id)
				if err != nil {
					_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "skip %s: %v\n", id, err)
					continue
				}
				gi := gitinfo.Inspect(p.Path)
				if !gi.IsRepo {
					continue
				}
				if err := a.Repo.UpdateGitStatus(p.ID, gi.Branch, gi.Dirty); err != nil {
					_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "update %s: %v\n", p.Name, err)
					continue
				}
				fmt.Printf("%-24s %s", p.Name, gi.Branch)
				if gi.Dirty > 0 {
					fmt.Printf(" +%d", gi.Dirty)
				}
				fmt.Println()
			}
			return nil
		},
	}
}
