package cli

import (
	"github.com/spf13/cobra"

	"github.com/bal16/projx/internal/tui"
)

func newTuiCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "tui",
		Short: "Interactive TUI project picker",
		RunE: func(cmd *cobra.Command, args []string) error {
			a := requireApp()
			res, err := tui.Run(a.Repo)
			if err != nil {
				return err
			}
			if !res.Opened || res.Path == "" {
				return nil
			}
			return runOpen(a, res.Path, openParams{
				newWindow: res.NewWindow,
				editor:    res.Editor,
			})
		},
	}
}
