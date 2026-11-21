package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/bal16/projx/internal/config"
	"github.com/bal16/projx/internal/discovery"
	"github.com/bal16/projx/internal/gitinfo"
	"github.com/bal16/projx/internal/model"
)

type scanParams struct {
	roots    []string
	maxDepth int
	dryRun   bool
}

func runScan(a *App, roots []string, sp scanParams) error {
	r := sp.roots
	if len(r) == 0 {
		r = a.Config.ScanRoots
	}
	if len(roots) > 0 {
		r = append(r, roots...)
	}
	if len(r) == 0 {
		return fmt.Errorf("no scan roots; pass roots or set scan_roots in config")
	}
	maxDepth := sp.maxDepth
	if maxDepth <= 0 {
		maxDepth = a.Config.ScanMaxDepth
	}
	sc := &discovery.Scanner{Markers: a.Config.ScanMarkers, MaxDepth: maxDepth}
	found, err := sc.Scan(r)
	if err != nil {
		return err
	}
	added := 0
	for _, f := range found {
		if existing, _ := a.Repo.GetByPath(f.Path); existing != nil {
			continue
		}
		p := &model.Project{
			Name:   f.Name,
			Path:   f.Path,
			Editor: "", // leave empty to follow default_editor
			Tags:   []string{},
		}
		if gi := gitinfo.Inspect(f.Path); gi.IsRepo {
			p.GitBranch = gi.Branch
			p.GitStaged = gi.Staged
			p.GitModified = gi.Modified
			p.GitUntracked = gi.Untracked
			p.GitDeleted = gi.Deleted
			p.GitRenamed = gi.Renamed
			p.GitConflicted = gi.Conflicted
			p.GitDirty = gi.Dirty
		}
		if sp.dryRun {
			fmt.Printf("[dry-run] would add %s (%s)\n", p.Name, p.Path)
			continue
		}
		if err := a.Repo.Add(p); err != nil {
			fmt.Printf("skip %s: %v\n", f.Path, err)
			continue
		}
		added++
		fmt.Printf("added %s\n", p.Name)
	}
	fmt.Printf("scanned %d found, %d added\n", len(found), added)

	// persist roots/max-depth
	if len(roots) > 0 {
		merged := append([]string{}, a.Config.ScanRoots...)
		for _, rr := range roots {
			if !contains(merged, rr) {
				merged = append(merged, rr)
			}
		}
		a.Config.ScanRoots = merged
	}
	if maxDepth > 0 {
		a.Config.ScanMaxDepth = maxDepth
	}
	_ = config.Save(a.Config, a.Repo)
	return nil
}

func newScanCmd() *cobra.Command {
	var sp scanParams
	var roots []string
	cmd := &cobra.Command{
		Use:   "scan [roots...]",
		Short: "Auto-discover projects in directories",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runScan(requireApp(), args, scanParams{
				roots:    roots,
				maxDepth: sp.maxDepth,
				dryRun:   sp.dryRun,
			})
		},
	}
	cmd.Flags().StringSliceVar(&roots, "roots", nil, "roots to scan (overrides config)")
	cmd.Flags().IntVar(&sp.maxDepth, "max-depth", 0, "max scan depth (0 = config default)")
	cmd.Flags().BoolVar(&sp.dryRun, "dry-run", false, "show what would be added")
	return cmd
}

func contains(ss []string, v string) bool {
	for _, s := range ss {
		if s == v {
			return true
		}
	}
	return false
}
