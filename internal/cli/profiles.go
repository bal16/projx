package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"

	"github.com/spf13/cobra"

	"github.com/bal16/projx/internal/platform"
)

// VSCodeProfile represents a known VS Code profile.
type VSCodeProfile struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// listVSCodeProfiles reads profile names from VS Code's sync data and
// profile directories. Returns profiles sorted by name.
func listVSCodeProfiles() []VSCodeProfile {
	base, _ := platform.VSCodeBaseDir("code")
	base = filepath.Join(base, "User")

	// name map from sync data
	nameMap := map[string]string{}
	syncPath := filepath.Join(base, "sync", "profiles", "lastSyncprofiles.json")
	if data, err := os.ReadFile(syncPath); err == nil {
		var sync struct {
			SyncData struct {
				Content string `json:"content"`
			} `json:"syncData"`
		}
		if json.Unmarshal(data, &sync) == nil {
			var profiles []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			}
			if json.Unmarshal([]byte(sync.SyncData.Content), &profiles) == nil {
				for _, p := range profiles {
					nameMap[p.ID] = p.Name
				}
			}
		}
	}

	// profile directories
	profilesDir := filepath.Join(base, "profiles")
	entries, _ := os.ReadDir(profilesDir)
	var out []VSCodeProfile
	seen := map[string]bool{}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		id := e.Name()
		if id == "builtin" {
			continue
		}
		name := nameMap[id]
		if name == "" {
			name = id
		}
		out = append(out, VSCodeProfile{ID: id, Name: name})
		seen[id] = true
	}
	// include synced profiles not present as directories
	for id, name := range nameMap {
		if !seen[id] {
			out = append(out, VSCodeProfile{ID: id, Name: name})
		}
	}
	// default profile
	out = append(out, VSCodeProfile{ID: "__default__profile__", Name: "Default"})

	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func newProfilesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "profiles",
		Short: "List or create VS Code profiles",
	}
	list := &cobra.Command{
		Use:   "list",
		Short: "List known VS Code profiles",
		RunE: func(cmd *cobra.Command, args []string) error {
			profiles := listVSCodeProfiles()
			if len(profiles) == 0 {
				fmt.Println("no profiles found")
				return nil
			}
			for _, p := range profiles {
				fmt.Println(p.Name)
			}
			return nil
		},
	}
	create := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a new VS Code profile (opens VS Code with that profile)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			// VS Code creates the profile on first use of --profile.
			c := exec.Command("code", "--profile", name)
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr
			if err := c.Start(); err != nil {
				return fmt.Errorf("launch code: %w", err)
			}
			_ = c.Process.Release()
			fmt.Printf("launched VS Code with profile %q (profile will be created if new)\n", name)
			return nil
		},
	}
	jsonOut := false
	list.Flags().BoolVar(&jsonOut, "json", false, "output JSON")
	// wrap list to support json
	origRun := list.RunE
	list.RunE = func(cmd *cobra.Command, args []string) error {
		if jsonOut {
			profiles := listVSCodeProfiles()
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(profiles)
		}
		return origRun(cmd, args)
	}
	cmd.AddCommand(list, create)
	// default to list
	cmd.RunE = list.RunE
	return cmd
}
