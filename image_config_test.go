package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfiguredImageViewerUsesExplicitConfig(t *testing.T) {
	previous := configFile
	t.Cleanup(func() { configFile = previous })
	configFile = filepath.Join(t.TempDir(), "glow.yml")
	if err := os.WriteFile(configFile, []byte("imageViewer: /custom/bin/kittyview\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := configuredImageViewer()
	if err != nil || got != "/custom/bin/kittyview" {
		t.Fatalf("got %q, %v", got, err)
	}
	configFile = filepath.Join(t.TempDir(), "missing.yml")
	if _, err := configuredImageViewer(); err == nil {
		t.Fatal("expected error for missing explicit configuration")
	}
}
