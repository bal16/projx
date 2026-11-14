package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

func runWsCreate(a *App, name string, paths []string) error {
	resolved := []string{}
	for _, p := range paths {
		if strings.HasPrefix(p, "~/") {
			home, _ := os.UserHomeDir()
			p = filepath.Join(home, p[2:])
		} else if !filepath.IsAbs(p) {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			p = filepath.Join(cwd, p)
		}
		if info, err := os.Stat(p); err != nil {
			fmt.Fprintf(os.Stderr, "warning: path does not exist: %s\n", p)
		} else if !info.IsDir() {
			fmt.Fprintf(os.Stderr, "warning: not a directory: %s\n", p)
		}
		resolved = append(resolved, p)
	}
	if err := a.Repo.AddSet(name, resolved); err != nil {
		return err
	}
	fmt.Printf("set %q created with %d paths\n", name, len(resolved))
	return nil
}

func runWsList(a *App) error {
	sets, err := a.Repo.ListSets()
	if err != nil {
		return err
	}
	if len(sets) == 0 {
		fmt.Println("no sets")
		return nil
	}
	for _, s := range sets {
		fmt.Printf("%-20s %s\n", s.Name, strings.Join(s.Paths, ", "))
	}
	return nil
}

func runWsDelete(a *App, name string) error {
	if err := a.Repo.RemoveSet(name); err != nil {
		return err
	}
	fmt.Printf("deleted set %s\n", name)
	return nil
}

func runWsShow(a *App, name string) error {
	s, err := a.Repo.GetSet(name)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(s)
}

func newWsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ws",
		Short: "Manage workspace sets (open multiple projects together)",
	}
	create := &cobra.Command{
		Use:   "create <name> <path...>",
		Short: "Create or update a workspace set",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWsCreate(requireApp(), args[0], args[1:])
		},
	}
	list := &cobra.Command{
		Use:   "list",
		Short: "List workspace sets",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWsList(requireApp())
		},
	}
	del := &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete a workspace set",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWsDelete(requireApp(), args[0])
		},
	}
	show := &cobra.Command{
		Use:   "show <name>",
		Short: "Show a workspace set as JSON",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWsShow(requireApp(), args[0])
		},
	}
	cmd.AddCommand(create, list, del, show)
	return cmd
}
