package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func newTagCmd() *cobra.Command {
	cmd := &cobra.Command{
		DisableFlagParsing: true,
		Use:                "tag <project> [+tag|-tag ...]",
		Short:              "Add or remove tags on a project",
		Args:               cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			a := requireApp()
			p, err := a.Repo.Get(args[0])
			if err != nil {
				return err
			}
			for _, t := range args[1:] {
				t = strings.TrimSpace(t)
				if strings.HasPrefix(t, "+") {
					p.AddTag(strings.TrimPrefix(t, "+"))
				} else if strings.HasPrefix(t, "-") {
					p.RemoveTag(strings.TrimPrefix(t, "-"))
				} else {
					p.AddTag(t)
				}
			}
			if err := a.Repo.Update(p); err != nil {
				return err
			}
			fmt.Printf("tags: %v\n", p.Tags)
			return nil
		},
	}
	return cmd
}
