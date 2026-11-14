package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func runTag(a *App, name string, tags []string) error {
	p, err := a.Repo.Get(name)
	if err != nil {
		return err
	}
	for _, t := range tags {
		t = strings.TrimSpace(t)
		switch {
		case strings.HasPrefix(t, "+"):
			p.AddTag(strings.TrimPrefix(t, "+"))
		case strings.HasPrefix(t, "-"):
			p.RemoveTag(strings.TrimPrefix(t, "-"))
		default:
			p.AddTag(t)
		}
	}
	if err := a.Repo.Update(p); err != nil {
		return err
	}
	fmt.Printf("tags: %v\n", p.Tags)
	return nil
}

func newTagCmd() *cobra.Command {
	cmd := &cobra.Command{
		DisableFlagParsing: true,
		Use:                "tag <project> [+tag|-tag ...]",
		Short:              "Add or remove tags on a project",
		Args:               cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTag(requireApp(), args[0], args[1:])
		},
	}
	return cmd
}
