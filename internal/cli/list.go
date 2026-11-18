package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/bal16/projx/internal/gitinfo"
	"github.com/bal16/projx/internal/model"
)

func newListCmd() *cobra.Command {
	var tag, group, sortKey string
	var refresh bool
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
			if refresh {
				for i := range projects {
					gi := gitinfo.Inspect(projects[i].Path)
					if gi.IsRepo {
						_ = a.Repo.UpdateGitStatus(projects[i].ID, gi)
						projects[i].GitBranch = gi.Branch
						projects[i].GitStaged = gi.Staged
						projects[i].GitModified = gi.Modified
						projects[i].GitUntracked = gi.Untracked
						projects[i].GitDeleted = gi.Deleted
						projects[i].GitRenamed = gi.Renamed
						projects[i].GitConflicted = gi.Conflicted
						projects[i].GitDirty = gi.Dirty
					}
				}
			}
			switch sortKey {
			case "name":
				// sort by name asc (favorites still first if not overridden)
			case "recent":
				// default: already sorted by repo (favorite desc, last_opened desc)
			case "path":
				// sort by path
			}
			// Apply sort (stable) on top of repo default.
			switch sortKey {
			case "name":
				// simple insertion sort to avoid import churn
				for i := 1; i < len(projects); i++ {
					j := i
					for j > 0 && projects[j-1].Name > projects[j].Name {
						projects[j-1], projects[j] = projects[j], projects[j-1]
						j--
					}
				}
			case "path":
				for i := 1; i < len(projects); i++ {
					j := i
					for j > 0 && projects[j-1].Path > projects[j].Path {
						projects[j-1], projects[j] = projects[j], projects[j-1]
						j--
					}
				}
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
				// Fields (0-9 stable): id, name, path, editor, tags, group, favorite, git_branch, git_dirty, remote
				// Appended: git_staged, git_modified, git_untracked, git_deleted, git_renamed, git_conflicted,
				//           gh_enabled, gh_open_prs, gh_open_issues, gh_fetched_at(unix)
				sep := "\x1f"
				for _, p := range projects {
					tags := joinStrings(p.Tags, ",")
					fav := "0"
					if p.Favorite {
						fav = "1"
					}
					ghEnabled := ""
					if p.GHEnabled != nil {
						if *p.GHEnabled {
							ghEnabled = "1"
						} else {
							ghEnabled = "0"
						}
					}
					ghPRs, ghIssues, ghFetched := "", "", ""
					if p.GHOpenPRs != nil {
						ghPRs = fmt.Sprintf("%d", *p.GHOpenPRs)
					}
					if p.GHOpenIssues != nil {
						ghIssues = fmt.Sprintf("%d", *p.GHOpenIssues)
					}
					if p.GHFetchedAt != nil {
						ghFetched = fmt.Sprintf("%d", p.GHFetchedAt.Unix())
					}
					fields := []string{
						p.ID, p.Name, p.Path, p.Editor, tags, p.Group, fav,
						p.GitBranch, fmt.Sprintf("%d", p.GitDirty), p.Remote,
						fmt.Sprintf("%d", p.GitStaged), fmt.Sprintf("%d", p.GitModified), fmt.Sprintf("%d", p.GitUntracked),
						fmt.Sprintf("%d", p.GitDeleted), fmt.Sprintf("%d", p.GitRenamed), fmt.Sprintf("%d", p.GitConflicted),
						ghEnabled, ghPRs, ghIssues, ghFetched,
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
						git = "  [" + p.GitBranch
						if detail := compactGitDetail(p); detail != "" {
							git += " " + detail
						}
						git += "]"
					}
					gh := ""
					if p.GHEnabled != nil && *p.GHEnabled && p.GHOpenPRs != nil && p.GHOpenIssues != nil {
						gh = fmt.Sprintf("  PR:%d I:%d", *p.GHOpenPRs, *p.GHOpenIssues)
					}
					tags := ""
					if len(p.Tags) > 0 {
						tags = "  tags=" + joinStrings(p.Tags, ",")
					}
					fmt.Printf("%s %-24s %s%s%s%s\n", marker, p.Name, p.Path, git, gh, tags)
				}
				return nil
			}
		},
	}
	cmd.Flags().StringVar(&tag, "tag", "", "filter by tag")
	cmd.Flags().StringVar(&group, "group", "", "filter by group")
	cmd.Flags().BoolVar(&favoritesOnly, "favorites", false, "only favorites")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "output JSON (shorthand for --format=json)")
	cmd.Flags().StringVar(&sortKey, "sort", "recent", "sort order: recent|name|path")
	cmd.Flags().BoolVar(&refresh, "refresh", false, "refresh cached git status before listing")
	cmd.Flags().StringVar(&format, "format", "text", "output format: text|json|delimited")
	return cmd
}

func compactGitDetail(p model.Project) string {
	var parts []string
	if p.GitStaged > 0 {
		parts = append(parts, fmt.Sprintf("S:%d", p.GitStaged))
	}
	if p.GitModified > 0 {
		parts = append(parts, fmt.Sprintf("M:%d", p.GitModified))
	}
	if p.GitUntracked > 0 {
		parts = append(parts, fmt.Sprintf("?:%d", p.GitUntracked))
	}
	if p.GitDeleted > 0 {
		parts = append(parts, fmt.Sprintf("D:%d", p.GitDeleted))
	}
	if p.GitRenamed > 0 {
		parts = append(parts, fmt.Sprintf("R:%d", p.GitRenamed))
	}
	if p.GitConflicted > 0 {
		parts = append(parts, fmt.Sprintf("U:%d", p.GitConflicted))
	}
	return strings.Join(parts, " ")
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
