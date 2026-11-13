package cli

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/bal16/projx/internal/tui"
	"github.com/spf13/cobra"
)

func newTuiCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "tui",
		Short: "Interactive TUI project picker",
		RunE: func(cmd *cobra.Command, args []string) error {
			a := requireApp()
			path, newWindow, err := tui.Run(a.Repo)
			if err != nil {
				return err
			}
			if path == "" {
				return nil
			}
			p, _ := a.Repo.GetByPath(path)

			ed := a.Config.DefaultEditor
			if p != nil && p.Editor != "" {
				ed = p.Editor
			}
			bin, ok := a.Config.EditorCommands[ed]
			if !ok {
				bin = ed
			}

			argv := []string{path}
			if newWindow {
				switch ed {
				case "zed", "code", "codium", "cursor":
					argv = []string{"-n", path}
				}
			}

			c := exec.Command(bin, argv...)
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr
			if err := c.Start(); err != nil {
				return fmt.Errorf("launch %s: %w", bin, err)
			}
			_ = c.Process.Release()
			if p != nil {
				_ = a.Repo.TouchLastOpened(p.ID)
			}
					fmt.Printf("opened %s in %s\n", path, ed)
			return nil
		},
	}
}
