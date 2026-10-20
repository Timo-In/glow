package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProjectCommandsManageConfiguredFile(t *testing.T) {
	dir := t.TempDir()
	projectPath := filepath.Join(dir, "docs")
	if err := os.Mkdir(projectPath, 0o755); err != nil {
		t.Fatal(err)
	}
	previousConfigFile := configFile
	configFile = filepath.Join(dir, "config", "glow.yml")
	t.Cleanup(func() { configFile = previousConfigFile })

	var output bytes.Buffer
	projectAddCmd.SetOut(&output)
	t.Cleanup(func() {
		projectAddCmd.SetOut(nil)
		projectListCmd.SetOut(nil)
		projectRemoveCmd.SetOut(nil)
	})
	if err := projectAddCmd.RunE(projectAddCmd, []string{"Docs", projectPath}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), `Added project "Docs"`) {
		t.Fatalf("unexpected add output %q", output.String())
	}

	output.Reset()
	projectListCmd.SetOut(&output)
	if err := projectListCmd.RunE(projectListCmd, nil); err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(output.String()); got != "Docs\t"+projectPath {
		t.Fatalf("unexpected list output %q", got)
	}

	names, directive := completeProjectNames(nil, nil, "")
	if len(names) != 1 || names[0] != "Docs" {
		t.Fatalf("unexpected completions %v", names)
	}
	if directive == 0 {
		t.Fatal("expected completion directive to disable file completion")
	}

	output.Reset()
	projectRemoveCmd.SetOut(&output)
	if err := projectRemoveCmd.RunE(projectRemoveCmd, []string{"Docs"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), `Removed project "Docs"`) {
		t.Fatalf("unexpected remove output %q", output.String())
	}

	output.Reset()
	if err := projectListCmd.RunE(projectListCmd, nil); err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(output.String()); got != "No projects configured." {
		t.Fatalf("unexpected empty list output %q", got)
	}
}

func TestProjectOpenRejectsMissingDirectory(t *testing.T) {
	dir := t.TempDir()
	previousConfigFile := configFile
	configFile = filepath.Join(dir, "glow.yml")
	t.Cleanup(func() { configFile = previousConfigFile })
	contents := `projects:
  - name: Missing
    path: /path/that/does/not/exist
`
	if err := os.WriteFile(configFile, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}

	err := projectOpenCmd.RunE(projectOpenCmd, []string{"Missing"})
	if err == nil || !strings.Contains(err.Error(), "unable to access project directory") {
		t.Fatalf("expected missing-directory error, got %v", err)
	}
}

func TestProjectSelectorRequiresConfiguredProject(t *testing.T) {
	dir := t.TempDir()
	previousConfigFile := configFile
	configFile = filepath.Join(dir, "glow.yml")
	t.Cleanup(func() { configFile = previousConfigFile })

	err := projectCmd.RunE(projectCmd, nil)
	if err == nil || !strings.Contains(err.Error(), "no projects configured") {
		t.Fatalf("expected no-projects error, got %v", err)
	}
}
