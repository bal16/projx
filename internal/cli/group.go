package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func runGroup(a *App, name, group string) error {
	p, err := a.Repo.Get(name)
	if err != nil {
		return err
	}
	if group == "-" || group == "none" {
		p.Group = ""
	} else {
		p.Group = group
	}
	if err := a.Repo.Update(p); err != nil {
		return err
	}
	fmt.Printf("group set to %q\n", p.Group)
	return nil
}

func newGroupCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "group <project> <group-name>",
		Short: "Set group for a project (use empty string to clear)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGroup(requireApp(), args[0], args[1])
		},
	}
}
