package cli

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/bal16/projx/internal/config"
)

func runConfigShow(a *App) error {
	b, _ := json.MarshalIndent(a.Config, "", "  ")
	fmt.Println(string(b))
	return nil
}

func runConfigGet(a *App, key string) error {
	switch key {
	case "scan_roots":
		b, _ := json.Marshal(a.Config.ScanRoots)
		fmt.Println(string(b))
	case "default_editor":
		fmt.Println(a.Config.DefaultEditor)
	case "scan_max_depth":
		fmt.Println(a.Config.ScanMaxDepth)
	case "editor_profiles":
		b, _ := json.Marshal(a.Config.EditorProfiles)
		fmt.Println(string(b))
	case "editor_commands":
		b, _ := json.Marshal(a.Config.EditorCommands)
		fmt.Println(string(b))
	case "terminal":
		fmt.Println(a.Config.Terminal)
	default:
		return fmt.Errorf("unknown key: %s", key)
	}
	return nil
}

func configKeys() []string {
	return []string{
		"scan_roots\tJSON array of root directories to scan",
		"default_editor\tDefault editor (e.g. zed, code, nvim)",
		"scan_max_depth\tMax directory depth for scan",
		"editor_profiles\tJSON object mapping editor -> default profile",
		"editor_commands\tJSON object mapping editor -> binary/command",
		"terminal\tTerminal emulator for TUI editors (empty = auto-detect)",
	}
}

func configSetKeys(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) != 0 {
		// completing <value>
		switch args[0] {
		case "default_editor":
			return []string{"zed", "code", "nvim", "vim", "cursor", "codium", "windsurf"}, cobra.ShellCompDirectiveNoFileComp
		case "terminal":
			return []string{"kitty", "alacritty", "foot", "wezterm", "ghostty", "konsole", "gnome-terminal", "xterm"}, cobra.ShellCompDirectiveNoFileComp
		case "scan_max_depth":
			return []string{"1", "2", "3", "4", "5"}, cobra.ShellCompDirectiveNoFileComp
		default:
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
	}
	keys := configKeys()
	// suggest dotted editor_commands.<name> entries
	editors := []string{"zed", "code", "nvim", "vim", "cursor", "codium", "helix", "hx"}
	for _, e := range editors {
		keys = append(keys, "editor_commands."+e+"\tSet binary for "+e)
	}
	return keys, cobra.ShellCompDirectiveNoFileComp
}

func configGetKeys(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) != 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return configKeys(), cobra.ShellCompDirectiveNoFileComp
}

func runConfigSet(a *App, key, value string) error {
	switch {
	case key == "editor_commands":
		var m map[string]string
		if err := json.Unmarshal([]byte(value), &m); err != nil {
			return fmt.Errorf("editor_commands must be JSON object: %w", err)
		}
		if a.Config.EditorCommands == nil {
			a.Config.EditorCommands = map[string]string{}
		}
		for k, v := range m {
			a.Config.EditorCommands[k] = v
		}
	case strings.HasPrefix(key, "editor_commands."):
		editor := strings.TrimPrefix(key, "editor_commands.")
		if editor == "" {
			return fmt.Errorf("editor_commands.<editor> requires an editor name")
		}
		if a.Config.EditorCommands == nil {
			a.Config.EditorCommands = map[string]string{}
		}
		a.Config.EditorCommands[editor] = value
	case key == "scan_roots":
		var roots []string
		if err := json.Unmarshal([]byte(value), &roots); err != nil {
			return fmt.Errorf("scan_roots must be JSON array: %w", err)
		}
		a.Config.ScanRoots = roots
	case key == "default_editor":
		a.Config.DefaultEditor = value
	case key == "terminal":
		a.Config.Terminal = value
	case key == "scan_max_depth":
		var n int
		if _, err := fmt.Sscanf(value, "%d", &n); err != nil {
			return err
		}
		a.Config.ScanMaxDepth = n
	case key == "editor_profiles":
		var m map[string]string
		if err := json.Unmarshal([]byte(value), &m); err != nil {
			return fmt.Errorf("editor_profiles must be JSON object: %w", err)
		}
		if a.Config.EditorProfiles == nil {
			a.Config.EditorProfiles = map[string]string{}
		}
		for k, v := range m {
			a.Config.EditorProfiles[k] = v
		}
	default:
		return fmt.Errorf("unknown key: %s", key)
	}
	return config.Save(a.Config, a.Repo)
}

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Show or set configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigShow(requireApp())
		},
	}
	get := &cobra.Command{
		Use:               "get <key>",
		Short:             "Get a config value (scan_roots, default_editor, scan_max_depth, editor_profiles, editor_commands, terminal)",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: configGetKeys,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigGet(requireApp(), args[0])
		},
	}
	set := &cobra.Command{
		Use:               "set <key> <value>",
		Short:             "Set a config value",
		Args:              cobra.ExactArgs(2),
		ValidArgsFunction: configSetKeys,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigSet(requireApp(), args[0], args[1])
		},
	}
	cmd.AddCommand(get, set)
	return cmd
}
