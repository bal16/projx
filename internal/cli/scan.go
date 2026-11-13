package cli

import (
	"fmt"

	"github.com/bal16/projx/internal/config"
	"github.com/bal16/projx/internal/discovery"
	"github.com/bal16/projx/internal/gitinfo"
	"github.com/bal16/projx/internal/model"
	"github.com/spf13/cobra"
)

func newScanCmd() *cobra.Command {
	var roots []string
	var maxDepth int
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "scan [roots...]",
		Short: "Auto-discover projects in directories",
		RunE: func(cmd *cobra.Command, args []string) error {
			a := requireApp()
			if len(roots) == 0 {
				roots = a.Config.ScanRoots
			}
			if len(args) > 0 {
				roots = append(roots, args...)
			}
			if len(roots) == 0 {
				return fmt.Errorf("no scan roots; pass roots or set scan_roots in config")
			}
			if maxDepth <= 0 {
				maxDepth = a.Config.ScanMaxDepth
			}
			sc := &discovery.Scanner{Markers: a.Config.ScanMarkers, MaxDepth: maxDepth}
			found, err := sc.Scan(roots)
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
					Editor: a.Config.DefaultEditor,
					Tags:   []string{},
				}
				if gi := gitinfo.Inspect(f.Path); gi.IsRepo {
					p.GitBranch = gi.Branch
					p.GitDirty = gi.Dirty
				}
				if dryRun {
					fmt.Printf("[dry-run] would add %s (%s)\n", p.Name, p.Path)
					continue
				}
				if err := a.Repo.Add(p); err != nil {
					fmt.Fprintf(cmd.ErrOrStderr(), "skip %s: %v\n", f.Path, err)
					continue
				}
				added++
				fmt.Printf("added %s\n", p.Name)
			}
			fmt.Printf("scanned %d found, %d added\n", len(found), added)
			// persist roots if new
			if len(args) > 0 {
				merged := append([]string{}, a.Config.ScanRoots...)
				for _, r := range args {
					if !contains(merged, r) {
						merged = append(merged, r)
					}
				}
				a.Config.ScanRoots = merged
				a.Config.ScanMaxDepth = maxDepth
				_ = config.Save(a.Config, a.Repo)
			}
			return nil
		},
	}
	cmd.Flags().StringSliceVar(&roots, "roots", nil, "roots to scan (overrides config)")
	cmd.Flags().IntVar(&maxDepth, "max-depth", 0, "max scan depth (0 = config default)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "show what would be added")
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
