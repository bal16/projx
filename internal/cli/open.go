package cli

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

func newOpenCmd() *cobra.Command {
	var editor, profile string
	var newWindow, addToWorkspace bool

	cmd := &cobra.Command{
		Use:   "open <project>",
		Short: "Open a project in an editor",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a := requireApp()
			p, err := a.Repo.Get(args[0])
			if err != nil {
				return err
			}

			ed := editor
			if ed == "" {
				ed = p.Editor
			}
			if ed == "" {
				ed = a.Config.DefaultEditor
			}
			prof := profile
			if prof == "" {
				prof = p.Profile
			}
			if prof == "" && a.Config.EditorProfiles != nil {
				prof = a.Config.EditorProfiles[ed]
			}
			bin, ok := a.Config.EditorCommands[ed]
			if !ok {
				bin = ed
			}

			var argv []string
			vscodeVariants := map[string]bool{"code": true, "codium": true, "cursor": true, "windsurf": true, "antigravity": true}
			switch {
			case newWindow:
				switch ed {
				case "zed":
					argv = []string{"-n", p.Path}
				case "code", "codium", "cursor", "windsurf", "antigravity":
					argv = []string{"-n"}
					if prof != "" {
						argv = append(argv, "--profile", prof)
					}
					argv = append(argv, p.Path)
				default:
					argv = []string{p.Path}
				}
			case addToWorkspace:
				switch ed {
				case "zed":
					argv = []string{"-a", p.Path}
				default:
					argv = []string{p.Path}
				}
			default:
				if vscodeVariants[ed] && prof != "" {
					argv = []string{"--profile", prof, p.Path}
				} else {
					argv = []string{p.Path}
				}
			}

			if p.Remote != "" && ed == "zed" {
				// remote project: zed ssh://...
				argv = []string{p.Path}
			}

			c := exec.Command(bin, argv...)
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr
			if err := c.Start(); err != nil {
				return fmt.Errorf("launch %s: %w", bin, err)
			}
			// detach
			_ = c.Process.Release()
			_ = a.Repo.TouchLastOpened(p.ID)
			fmt.Printf("opened %s in %s\n", p.Name, ed)
			return nil
		},
	}
	cmd.Flags().StringVar(&editor, "editor", "", "override editor")
	cmd.Flags().BoolVar(&newWindow, "new-window", false, "open in a new window")
	cmd.Flags().BoolVar(&addToWorkspace, "add", false, "add to current workspace/window")
	cmd.Flags().StringVar(&profile, "profile", "", "override editor profile (VS Code)")
	return cmd
}
