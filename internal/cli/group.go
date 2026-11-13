package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newGroupCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "group <project> <group-name>",
		Short: "Set group for a project (use empty string to clear)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			a := requireApp()
			p, err := a.Repo.Get(args[0])
			if err != nil {
				return err
			}
			if args[1] == "-" || args[1] == "none" {
				p.Group = ""
			} else {
				p.Group = args[1]
			}
			if err := a.Repo.Update(p); err != nil {
				return err
			}
			fmt.Printf("group set to %q\n", p.Group)
			return nil
		},
	}
}
