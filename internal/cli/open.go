package cli

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

type openParams struct {
	editor         string
	profile        string
	setName        string
	newWindow      bool
	addToWorkspace bool
}

func runOpen(a *App, name string, op openParams) error {
	if name == "" && op.setName == "" {
		return fmt.Errorf("provide a project or --set <name>")
	}
	if op.setName != "" {
		st, err := a.Repo.GetSet(op.setName)
		if err != nil {
			return err
		}
		ed := op.editor
		if ed == "" {
			ed = a.Config.DefaultEditor
		}
		bin, ok := a.Config.EditorCommands[ed]
		if !ok {
			bin = ed
		}
		argv := append([]string{}, st.Paths...)
		if op.newWindow {
			argv = append([]string{"-n"}, argv...)
		}
		c := exec.Command(bin, argv...)
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		if err := c.Start(); err != nil {
			return fmt.Errorf("launch %s: %w", bin, err)
		}
		_ = c.Process.Release()
		fmt.Printf("opened set %s (%d projects) in %s\n", op.setName, len(st.Paths), ed)
		return nil
	}
	p, err := a.Repo.Get(name)
	if err != nil {
		return err
	}
	ed := op.editor
	if ed == "" {
		ed = p.Editor
	}
	if ed == "" {
		ed = a.Config.DefaultEditor
	}
	prof := op.profile
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
	vscodeVariants := map[string]bool{"code": true, "codium": true, "cursor": true, "windsurf": true, "antigravity": true}
	var argv []string
	switch {
	case op.newWindow:
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
	case op.addToWorkspace:
		switch ed {
		case "zed":
			argv = []string{"-a", p.Path}
		case "code", "codium", "cursor", "windsurf", "antigravity":
			argv = []string{"--add", p.Path}
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
	c := exec.Command(bin, argv...)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	if err := c.Start(); err != nil {
		return fmt.Errorf("launch %s: %w", bin, err)
	}
	_ = c.Process.Release()
	_ = a.Repo.TouchLastOpened(p.ID)
	fmt.Printf("opened %s in %s\n", p.Path, ed)
	return nil
}

func newOpenCmd() *cobra.Command {
	var op openParams
	cmd := &cobra.Command{
		Use:   "open <project>",
		Short: "Open a project in an editor",
		Args:  cobra.RangeArgs(0, 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := ""
			if len(args) > 0 {
				name = args[0]
			}
			return runOpen(requireApp(), name, op)
		},
	}
	cmd.Flags().StringVar(&op.editor, "editor", "", "override editor")
	cmd.Flags().StringVar(&op.profile, "profile", "", "override editor profile (VS Code)")
	cmd.Flags().StringVar(&op.setName, "set", "", "open a workspace set by name")
	cmd.Flags().BoolVar(&op.newWindow, "new-window", false, "open in a new window")
	cmd.Flags().BoolVar(&op.addToWorkspace, "add", false, "add to current workspace/window")
	return cmd
}
