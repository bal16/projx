package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func newListCmd() *cobra.Command {
	var tag, group string
	var favoritesOnly, jsonOut bool
	var format string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List projects",
		RunE: func(cmd *cobra.Command, args []string) error {
			a := requireApp()
			projects, err := a.Repo.List(tag, group, favoritesOnly)
			if err != nil {
				return err
			}
			if jsonOut {
				format = "json"
			}
			if format == "" {
				format = "text"
			}
			switch format {
			case "json":
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(projects)
			case "delimited":
				// Unit separator (\x1f) between fields, newline between records.
				// Fields: id, name, path, editor, tags, group, favorite, git_branch, git_dirty, remote
				sep := "\x1f"
				for _, p := range projects {
					tags := joinStrings(p.Tags, ",")
					fav := "0"
					if p.Favorite {
						fav = "1"
					}
					fields := []string{
						p.ID, p.Name, p.Path, p.Editor, tags, p.Group, fav,
						p.GitBranch, fmt.Sprintf("%d", p.GitDirty), p.Remote,
					}
					fmt.Println(joinStrings(fields, sep))
				}
				return nil
			default:
				if len(projects) == 0 {
					fmt.Println("no projects")
					return nil
				}
				for _, p := range projects {
					marker := " "
					if p.Favorite {
						marker = "*"
					}
					git := ""
					if p.GitBranch != "" {
						git = fmt.Sprintf("  [%s", p.GitBranch)
						if p.GitDirty > 0 {
							git += fmt.Sprintf(" +%d", p.GitDirty)
						}
						git += "]"
					}
					tags := ""
					if len(p.Tags) > 0 {
						tags = "  tags=" + joinStrings(p.Tags, ",")
					}
					fmt.Printf("%s %-24s %s%s%s\n", marker, p.Name, p.Path, git, tags)
				}
				return nil
			}
		},
	}
	cmd.Flags().StringVar(&tag, "tag", "", "filter by tag")
	cmd.Flags().StringVar(&group, "group", "", "filter by group")
	cmd.Flags().BoolVar(&favoritesOnly, "favorites", false, "only favorites")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "output JSON (shorthand for --format=json)")
	cmd.Flags().StringVar(&format, "format", "text", "output format: text|json|delimited")
	return cmd
}

func joinStrings(ss []string, sep string) string {
	out := ""
	for i, s := range ss {
		if i > 0 {
			out += sep
		}
		out += s
	}
	return out
}
