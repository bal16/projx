package cli

import (
	"encoding/json"
	"fmt"

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
	default:
		return fmt.Errorf("unknown key: %s", key)
	}
	return nil
}

func runConfigSet(a *App, key, value string) error {
	switch key {
	case "scan_roots":
		var roots []string
		if err := json.Unmarshal([]byte(value), &roots); err != nil {
			return fmt.Errorf("scan_roots must be JSON array: %w", err)
		}
		a.Config.ScanRoots = roots
	case "default_editor":
		a.Config.DefaultEditor = value
	case "scan_max_depth":
		var n int
		if _, err := fmt.Sscanf(value, "%d", &n); err != nil {
			return err
		}
		a.Config.ScanMaxDepth = n
	case "editor_profiles":
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
		Use:   "get <key>",
		Short: "Get a config value (scan_roots, default_editor, scan_max_depth, editor_profiles)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigGet(requireApp(), args[0])
		},
	}
	set := &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a config value",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigSet(requireApp(), args[0], args[1])
		},
	}
	cmd.AddCommand(get, set)
	return cmd
}
