// Package cli implements the pjx command-line interface.
package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/bal16/projx/internal/config"
	"github.com/bal16/projx/internal/db"
)

// App holds shared state for commands.
type App struct {
	DB     *db.DB
	Repo   *db.Repository
	Config *config.Config
}

var app *App

// NewRootCommand builds the root cobra command.
func NewRootCommand() *cobra.Command {
	root := &cobra.Command{
		Use:     "pjx",
		Short:   "projx - universal project manager for developers",
		Version: "0.2.0",
		Long: `pjx manages your projects/workspaces with aliases, tags, groups and
git status, and launches them in your preferred editor (zed, vscode, nvim, ...).

Use 'pjx scan' to auto-discover git repositories, 'pjx import' to migrate
from an existing editor, and 'pjx list' (JSON) to feed a shell launcher.`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Skip DB init for commands that don't need it.
			if cmd.Name() == "completion" || cmd.Name() == "help" {
				return nil
			}
			d, err := db.Open()
			if err != nil {
				return fmt.Errorf("open database: %w", err)
			}
			repo := db.NewRepository(d)
			cfg, err := config.Load(repo)
			if err != nil {
				_ = d.Close()
				return fmt.Errorf("load config: %w", err)
			}
			app = &App{DB: d, Repo: repo, Config: cfg}
			return nil
		},
		PersistentPostRun: func(cmd *cobra.Command, args []string) {
			if app != nil && app.DB != nil {
				_ = app.DB.Close()
			}
		},
	}
	root.AddCommand(
		newAddCmd(),
		newListCmd(),
		newOpenCmd(),
		newRemoveCmd(),
		newRenameCmd(),
		newTagCmd(),
		newGroupCmd(),
		newFavoriteCmd(),
		newProfilesCmd(),
		newSetCmd(),
		newScanCmd(),
		newImportCmd(),
		newRefreshCmd(),
		newConfigCmd(),
		newDoctorCmd(),
		newExportCmd(),
		newWsCmd(),
		newImportFileCmd(),
		newTuiCmd(),
		newVersionCmd(),
	)
	return root
}

func newVersionCmd() *cobra.Command {
	var showPlugins bool
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("pjx 0.2.0")
			if showPlugins {
				fmt.Println("plugins:")
				fmt.Println("  noctalia: 0.2.0 (plugins/project-manager)")
				fmt.Println("  dms:      0.2.0 (plugins/dms)")
				fmt.Println("  dms-widget: 0.2.0 (plugins/dms-widget)")
			}
		},
	}
	cmd.Flags().BoolVar(&showPlugins, "plugins", false, "also print plugin versions")
	return cmd
}

// helper to require app
func requireApp() *App {
	if app == nil {
		fmt.Fprintln(os.Stderr, "pjx: not initialized")
		os.Exit(1)
	}
	return app
}
