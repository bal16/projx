package cli

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

func newOpenCmd() *cobra.Command {
	var editor, profile, setName string
	var newWindow, addToWorkspace bool

	cmd := &cobra.Command{
		Use:   "open <project>",
		Short: "Open a project in an editor",
		Args:  cobra.RangeArgs(0, 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a := requireApp()
			if len(args) == 0 && setName == "" {
				return fmt.Errorf("provide a project or --set <name>")
			}
			if setName != "" {
				st, err := a.Repo.GetSet(setName)
				if err != nil {
					return err
				}
				ed := editor
				if ed == "" {
					ed = a.Config.DefaultEditor
				}
				bin, ok := a.Config.EditorCommands[ed]
				if !ok {
					bin = ed
				}
				// open all paths in one invocation
				argv := append([]string{}, st.Paths...)
				if newWindow {
					argv = append([]string{"-n"}, argv...)
				}
				c := exec.Command(bin, argv...)
				c.Stdout = os.Stdout
				c.Stderr = os.Stderr
				if err := c.Start(); err != nil {
					return fmt.Errorf("launch %s: %w", bin, err)
				}
				_ = c.Process.Release()
				fmt.Printf("opened set %s (%d projects) in %s\n", setName, len(st.Paths), ed)
				return nil
			}
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

			// Remote (SSH) projects: zed handles ssh:// URIs with the same flags.

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
	cmd.Flags().StringVar(&setName, "set", "", "open a workspace set by name")
	return cmd
}
