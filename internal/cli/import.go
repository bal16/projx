package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/bal16/projx/internal/importer"
)

func newImportCmd() *cobra.Command {
	var all bool

	cmd := &cobra.Command{
		Use:   "import [zed|code|codium|cursor|windsurf|projectmanager|pm]",
		Short: "Import projects from an existing editor (one-time)",
		RunE: func(cmd *cobra.Command, args []string) error {
			a := requireApp()

			var importers []importer.Importer
			switch {
			case all:
				importers = importer.All()
			case len(args) > 0:
				for _, name := range args {
					imp, err := importer.ByName(name)
					if err != nil {
						return err
					}
					importers = append(importers, imp)
				}
			default:
				return fmt.Errorf("specify an editor or use --all")
			}

			total := 0
			for _, imp := range importers {
				if !imp.Available() {
					fmt.Printf("skip %s: not available\n", imp.Name())
					continue
				}
				projects, err := imp.Import()
				if err != nil {
					fmt.Printf("error importing %s: %v\n", imp.Name(), err)
					continue
				}
				added := 0
				for _, p := range projects {
					if existing, _ := a.Repo.GetByPath(p.Path); existing != nil {
						continue
					}
					pp := p
					if err := a.Repo.Add(&pp); err != nil {
						continue
					}
					added++
				}
				total += added
				fmt.Printf("imported from %s: %d new (%d total)\n", imp.Name(), added, len(projects))
			}
			fmt.Printf("done: %d new projects\n", total)
			return nil
		},
	}
	cmd.Flags().BoolVar(&all, "all", false, "import from all available editors")
	return cmd
}
