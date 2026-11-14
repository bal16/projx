package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func runFavorite(a *App, name string) error {
	p, err := a.Repo.Get(name)
	if err != nil {
		return err
	}
	p.Favorite = !p.Favorite
	if err := a.Repo.Update(p); err != nil {
		return err
	}
	state := "unfavorited"
	if p.Favorite {
		state = "favorited"
	}
	fmt.Printf("%s %s\n", p.Name, state)
	return nil
}

func newFavoriteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "favorite <project>",
		Short: "Toggle favorite status of a project",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runFavorite(requireApp(), args[0])
		},
	}
}
