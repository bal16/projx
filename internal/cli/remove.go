package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func runRemove(a *App, name string) error {
	if err := a.Repo.Remove(name); err != nil {
		return err
	}
	fmt.Printf("removed %s\n", name)
	return nil
}

func newRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove <project>",
		Short: "Remove a project",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRemove(requireApp(), args[0])
		},
	}
}
