package platform

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestVSCodeBaseDirLinux(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("test only for linux")
	}
	got, _ := VSCodeBaseDir("code")
	want := filepath.Join(os.Getenv("HOME"), ".config", "Code")
	if got != want {
		t.Errorf("VSCodeBaseDir = %s, want %s", got, want)
	}
}

func TestVSCodeBaseDirDarwin(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("test only for darwin")
	}
	got, _ := VSCodeBaseDir("code")
	want := filepath.Join(os.Getenv("HOME"), "Library", "Application Support", "Code")
	if got != want {
		t.Errorf("VSCodeBaseDir = %s, want %s", got, want)
	}
}

func TestVSCodeBaseDirWindows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("test only for windows")
	}
	got, _ := VSCodeBaseDir("code")
	if got == "" {
		t.Fatal("empty path")
	}
}

func TestZedDataDir(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test only for linux/darwin")
	}
	got, _ := ZedDataDir()
	if got == "" {
		t.Fatal("empty path")
	}
	if !filepath.IsAbs(got) {
		t.Errorf("not absolute path: %s", got)
	}
}

func TestZedDBPath(t *testing.T) {
	pp, err := ZedDBPath("stable")
	if err != nil {
		t.Fatal(err)
	}
	if pp == "" {
		t.Fatal("empty path")
	}
	if !filepath.IsAbs(pp) {
		t.Errorf("not absolute path: %s", pp)
	}
}

func TestVSCodeStateDB(t *testing.T) {
	pp, err := VSCodeStateDB("code")
	if err != nil {
		t.Fatal(err)
	}
	if pp == "" {
		t.Fatal("empty path")
	}
	if !filepath.IsAbs(pp) {
		t.Errorf("not absolute path: %s", pp)
	}
}

func TestVSCodeProjectManagerJSON(t *testing.T) {
	pp, err := VSCodeProjectManagerJSON("code")
	if err != nil {
		t.Fatal(err)
	}
	if pp == "" {
		t.Fatal("empty path")
	}
	if !filepath.IsAbs(pp) {
		t.Errorf("not absolute path: %s", pp)
	}
}

func TestZedDataDirWindows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("test only for windows")
	}
	got, _ := ZedDataDir()
	if got == "" {
		t.Fatal("empty path")
	}
}
