package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/bal16/projx/internal/model"
	"github.com/bal16/projx/internal/util"
)

func runExport(a *App, file string) error {
	projects, err := a.Repo.List("", "", false)
	if err != nil {
		return err
	}
	if file != "" {
		f, err := os.Create(file)
		if err != nil {
			return err
		}
		defer func() { _ = f.Close() }()
		enc := json.NewEncoder(f)
		enc.SetIndent("", "  ")
		if err := enc.Encode(projects); err != nil {
			return err
		}
		fmt.Printf("exported %d projects to %s\n", len(projects), file)
		return nil
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(projects)
}

func newExportCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "export [file]",
		Short: "Export projects to JSON (stdout if no file)",
		RunE: func(cmd *cobra.Command, args []string) error {
			file := ""
			if len(args) > 0 {
				file = args[0]
			}
			return runExport(requireApp(), file)
		},
	}
}

func runImportFile(a *App, file string) error {
	data, err := os.ReadFile(file)
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
		if p.Name == "" {
			p.Name = util.BaseName(p.Path)
		}
		if p.Tags == nil {
			p.Tags = []string{}
		}
		if err := a.Repo.Add(p); err != nil {
			continue
		}
		added++
	}
	fmt.Printf("imported %d new projects from %s\n", added, file)
	return nil
}

func newImportFileCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "import-file <file>",
		Short: "Import projects from a JSON file (export format)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runImportFile(requireApp(), args[0])
		},
	}
}
