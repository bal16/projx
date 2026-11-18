package cli

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/bal16/projx/internal/ghinfo"
	"github.com/bal16/projx/internal/gitinfo"
)

func runRefresh(a *App, names []string) error {
	list := names
	if len(list) == 0 {
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
			fmt.Fprintf(os.Stderr, "skip %s: %v\n", id, err)
			continue
		}
		gi := gitinfo.Inspect(p.Path)
		if gi.IsRepo {
			if err := a.Repo.UpdateGitStatus(p.ID, gi); err != nil {
				fmt.Fprintf(os.Stderr, "update %s: %v\n", p.Name, err)
			}
		}
		if p.GHEnabled != nil && *p.GHEnabled {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			if c, err := ghinfo.FetchOpenCounts(ctx, p.Path); err == nil {
				_ = a.Repo.UpdateGitHubStatus(p.ID, c.OpenPRs, c.OpenIssues, c.FetchedAt)
			}
			cancel()
		}
		if gi.IsRepo {
			fmt.Printf("%-24s %s", p.Name, gi.Branch)
			if gi.Dirty > 0 {
				fmt.Printf(" +%d", gi.Dirty)
			}
			fmt.Println()
		} else if p.GHEnabled != nil && *p.GHEnabled {
			fmt.Printf("%-24s (github refreshed)\n", p.Name)
		}
	}
	return nil
}

func newRefreshCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "refresh [project]",
		Short: "Refresh cached git status for projects",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRefresh(requireApp(), args)
		},
	}
}
