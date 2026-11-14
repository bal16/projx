// Package platform provides OS-aware default paths for supported editors.
package platform

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// vscodeBaseDir returns the base user data directory for a VS Code variant.
// variant: "code", "codium", "cursor", "windsurf"
func VSCodeBaseDir(variant string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dirName := map[string]string{
		"code":     "Code",
		"codium":   "VSCodium",
		"cursor":   "Cursor",
		"windsurf": "Windsurf",
	}[variant]
	if dirName == "" {
		dirName = "Code"
	}
	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(home, "Library", "Application Support", dirName), nil
	case "windows":
		if ad := os.Getenv("APPDATA"); ad != "" {
			return filepath.Join(ad, dirName), nil
		}
		return filepath.Join(home, "AppData", "Roaming", dirName), nil
	default: // linux and others
		return filepath.Join(home, ".config", dirName), nil
	}
}

// VSCodeStateDB returns the path to state.vscdb for a variant.
func VSCodeStateDB(variant string) (string, error) {
	base, err := VSCodeBaseDir(variant)
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "User", "globalStorage", "state.vscdb"), nil
}

// VSCodeProjectManagerJSON returns the path to the Project Manager extension's projects.json.
func VSCodeProjectManagerJSON(variant string) (string, error) {
	base, err := VSCodeBaseDir(variant)
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "User", "globalStorage", "alefragnani.project-manager", "projects.json"), nil
}

// ZedDataDir returns the base data directory for Zed.
func ZedDataDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(home, "Library", "Application Support", "Zed"), nil
	case "windows":
		if ld := os.Getenv("LOCALAPPDATA"); ld != "" {
			return filepath.Join(ld, "Zed"), nil
		}
		return filepath.Join(home, "AppData", "Local", "Zed"), nil
	default:
		return filepath.Join(home, ".local", "share", "zed"), nil
	}
}

// ZedDBPath returns the path to Zed's workspace db.sqlite.
func ZedDBPath(channel string) (string, error) {
	if channel == "" {
		channel = "stable"
	}
	base, err := ZedDataDir()
	if err != nil {
		return "", err
	}
	// channel dir naming: 0-stable, 0-preview
	return filepath.Join(base, "db", "0-"+channel, "db.sqlite"), nil
}

// VSCodeProfilesDir returns the profiles directory for a variant.
func VSCodeProfilesDir(variant string) (string, error) {
	base, err := VSCodeBaseDir(variant)
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "User", "profiles"), nil
}

// VSCodeSyncProfilesPath returns the sync profiles json path.
func VSCodeSyncProfilesPath(variant string) (string, error) {
	base, err := VSCodeBaseDir(variant)
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "User", "sync", "profiles", "lastSyncprofiles.json"), nil
}

func String() string { return fmt.Sprintf("GOOS=%s", runtime.GOOS) }
