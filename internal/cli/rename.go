package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func runRename(a *App, name, newName string) error {
	p, err := a.Repo.Get(name)
	if err != nil {
		return err
	}
	p.Name = newName
	if err := a.Repo.Update(p); err != nil {
		return err
	}
	fmt.Printf("renamed to %s\n", p.Name)
	return nil
}

func newRenameCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rename <project> <new-name>",
		Short: "Rename a project",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRename(requireApp(), args[0], args[1])
		},
	}
}
