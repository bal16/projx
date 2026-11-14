package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/bal16/projx/internal/model"
	"github.com/spf13/cobra"
)

func newExportCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "export [file]",
		Short: "Export projects to JSON (stdout if no file)",
		RunE: func(cmd *cobra.Command, args []string) error {
			a := requireApp()
			projects, err := a.Repo.List("", "", false)
			if err != nil {
				return err
			}
			if len(args) > 0 {
				f, err := os.Create(args[0])
				if err != nil {
					return err
				}
				defer f.Close()
				enc := json.NewEncoder(f)
				enc.SetIndent("", "  ")
				if err := enc.Encode(projects); err != nil {
					return err
				}
				fmt.Printf("exported %d projects to %s\n", len(projects), args[0])
				return nil
			}
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(projects)
		},
	}
}

func newImportFileCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "import-file <file>",
		Short: "Import projects from a JSON file (export format)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a := requireApp()
			data, err := os.ReadFile(args[0])
			if err != nil {
				return err
			}
			var projects []model.Project
			if err := json.Unmarshal(data, &projects); err != nil {
				return err
			}
			added := 0
			for i := range projects {
				p := &projects[i]
				if p.Path == "" {
					continue
				}
				if existing, _ := a.Repo.GetByPath(p.Path); existing != nil {
					continue
				}
				if p.ID == "" {
					p.ID = "" // let Add generate
				}
				if p.Name == "" {
					p.Name = baseName(p.Path)
				}
				if p.Tags == nil {
					p.Tags = []string{}
				}
				if err := a.Repo.Add(p); err != nil {
					continue
				}
				added++
			}
			fmt.Printf("imported %d new projects from %s\n", added, args[0])
			return nil
		},
	}
}

func baseName(p string) string {
	parts := splitPath(p)
	if len(parts) == 0 {
		return p
	}
	return parts[len(parts)-1]
}

func splitPath(p string) []string {
	// simple split
	var out []string
	cur := ""
	for _, c := range p {
		if c == '/' {
			if cur != "" {
				out = append(out, cur)
			}
			cur = ""
		} else {
			cur += string(c)
		}
	}
	if cur != "" {
		out = append(out, cur)
	}
	return out
}
