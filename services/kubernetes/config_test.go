package services_kubernetes

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A literal "~/.kube/config" reached os.Stat unexpanded, so --clean silently
// did nothing and a custom path was never honoured.
func TestExpandPathResolvesTilde(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	got, err := ExpandPath("~/.kube/config")
	if err != nil {
		t.Fatalf("ExpandPath: %v", err)
	}
	if strings.Contains(got, "~") {
		t.Errorf("ExpandPath left a tilde in %q", got)
	}
	if want := filepath.Join(home, ".kube", "config"); got != want {
		t.Errorf("ExpandPath = %q, want %q", got, want)
	}
}

func TestExpandPathDefaultsAndPassesThrough(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	got, err := ExpandPath("")
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(home, ".kube", "config"); got != want {
		t.Errorf("empty path = %q, want %q", got, want)
	}

	abs := filepath.Join(t.TempDir(), "custom")
	if got, _ := ExpandPath(abs); got != abs {
		t.Errorf("absolute path = %q, want %q", got, abs)
	}
}

func TestBackupKubeconfigKeepsPreviousBackups(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config")
	if err := os.WriteFile(path, []byte("first"), 0600); err != nil {
		t.Fatal(err)
	}

	firstBackup, err := BackupKubeconfig(path)
	if err != nil {
		t.Fatalf("BackupKubeconfig: %v", err)
	}
	if data, _ := os.ReadFile(firstBackup); string(data) != "first" {
		t.Errorf("backup holds %q, want \"first\"", data)
	}
	if data, _ := os.ReadFile(path); len(data) != 0 {
		t.Errorf("kubeconfig should be emptied, got %q", data)
	}

	// A second run must not clobber the first backup.
	if err := os.WriteFile(path, []byte("second"), 0600); err != nil {
		t.Fatal(err)
	}
	secondBackup, err := BackupKubeconfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if secondBackup == firstBackup {
		t.Error("second backup overwrote the first")
	}
	if data, _ := os.ReadFile(firstBackup); string(data) != "first" {
		t.Error("the original backup was lost")
	}
}

func TestBackupKubeconfigMissingFile(t *testing.T) {
	backup, err := BackupKubeconfig(filepath.Join(t.TempDir(), "absent"))
	if err != nil {
		t.Fatalf("missing kubeconfig should not error: %v", err)
	}
	if backup != "" {
		t.Errorf("backup = %q, want empty", backup)
	}
}

func TestFlagValue(t *testing.T) {
	args := `["eks","get-token","--cluster-name","prod","--region","us-west-2"]`
	if got := flagValue(args, "--region"); got != "us-west-2" {
		t.Errorf("--region = %q, want us-west-2", got)
	}
	if got := flagValue(args, "--cluster-name"); got != "prod" {
		t.Errorf("--cluster-name = %q, want prod", got)
	}
	if got := flagValue(args, "--missing"); got != "" {
		t.Errorf("--missing = %q, want empty", got)
	}
	if got := flagValue("", "--region"); got != "" {
		t.Errorf("empty args = %q, want empty", got)
	}
}
