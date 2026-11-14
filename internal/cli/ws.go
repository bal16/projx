package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

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
			a := requireApp()
			name := args[0]
			paths := []string{}
			for _, p := range args[1:] {
				if strings.HasPrefix(p, "~/") {
					home, _ := os.UserHomeDir()
					p = home + "/" + p[2:]
				}
				abs, err := os.Getwd()
				_ = abs
				if err == nil {
					// make absolute if relative
					if !strings.HasPrefix(p, "/") {
						cwd, _ := os.Getwd()
						p = cwd + "/" + p
					}
				}
				paths = append(paths, p)
			}
			if err := a.Repo.AddSet(name, paths); err != nil {
				return err
			}
			fmt.Printf("set %q created with %d paths\n", name, len(paths))
			return nil
		},
	}
	list := &cobra.Command{
		Use:   "list",
		Short: "List workspace sets",
		RunE: func(cmd *cobra.Command, args []string) error {
			a := requireApp()
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
		},
	}
	del := &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete a workspace set",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a := requireApp()
			if err := a.Repo.RemoveSet(args[0]); err != nil {
				return err
			}
			fmt.Printf("deleted set %s\n", args[0])
			return nil
		},
	}
	show := &cobra.Command{
		Use:   "show <name>",
		Short: "Show a workspace set as JSON",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a := requireApp()
			s, err := a.Repo.GetSet(args[0])
			if err != nil {
				return err
			}
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(s)
		},
	}
	cmd.AddCommand(create, list, del, show)
	return cmd
}
