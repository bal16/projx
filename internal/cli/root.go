// Package cli implements the pjx command-line interface.
package cli

import (
	"fmt"
	"os"

	"github.com/bal16/projx/internal/config"
	"github.com/bal16/projx/internal/db"
	"github.com/spf13/cobra"
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
		Use:   "pjx",
		Short: "projx - universal project manager for developers",
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
				d.Close()
				return fmt.Errorf("load config: %w", err)
			}
			app = &App{DB: d, Repo: repo, Config: cfg}
			return nil
		},
		PersistentPostRun: func(cmd *cobra.Command, args []string) {
			if app != nil && app.DB != nil {
				app.DB.Close()
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
		newScanCmd(),
		newImportCmd(),
		newRefreshCmd(),
		newConfigCmd(),
		newVersionCmd(),
	)
	return root
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("pjx 0.1.0")
		},
	}
}

// helper to require app
func requireApp() *App {
	if app == nil {
		fmt.Fprintln(os.Stderr, "pjx: not initialized")
		os.Exit(1)
	}
	return app
}
