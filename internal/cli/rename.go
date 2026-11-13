package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newRenameCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rename <project> <new-name>",
		Short: "Rename a project",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			a := requireApp()
			p, err := a.Repo.Get(args[0])
			if err != nil {
				return err
			}
			p.Name = args[1]
			if err := a.Repo.Update(p); err != nil {
				return err
			}
			fmt.Printf("renamed to %s\n", p.Name)
			return nil
		},
	}
}
