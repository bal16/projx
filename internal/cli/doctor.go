package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func runDoctor(a *App, prune bool) error {
	projects, err := a.Repo.List("", "", false)
	if err != nil {
		return err
	}
	missing := 0
	for _, p := range projects {
		if _, err := os.Stat(p.Path); err != nil {
			missing++
			fmt.Printf("MISSING  %-24s %s\n", p.Name, p.Path)
			if prune {
				if err := a.Repo.Remove(p.ID); err != nil {
					fmt.Printf("  failed to remove: %v\n", err)
				} else {
					fmt.Printf("  removed\n")
				}
			}
		} else {
			fmt.Printf("OK       %-24s %s\n", p.Name, p.Path)
		}
	}
	fmt.Printf("\n%d missing out of %d projects\n", missing, len(projects))
	if missing > 0 && !prune {
		fmt.Println("run `pjx doctor --prune` to remove missing projects")
	}
	return nil
}

func newDoctorCmd() *cobra.Command {
	var prune bool
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Check for missing project paths and optionally prune them",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDoctor(requireApp(), prune)
		},
	}
	cmd.Flags().BoolVar(&prune, "prune", false, "remove projects whose path is missing")
	return cmd
}
