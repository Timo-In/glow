//go:build !windows

package ui

import (
	"context"
	"os"
	"os/exec"
	"testing"
	"time"
)

func TestImageViewerTerminalOutput(t *testing.T) {
	python, err := exec.LookPath("python3")
	if err != nil {
		t.Skip("python3 is required for the PTY regression test")
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, python, "testdata/image_viewer_terminal.py", executable, t.TempDir())
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("image viewer terminal regression: %v\n%s", err, output)
	}
}

func TestImageViewerTerminalHelper(t *testing.T) {
	viewer := os.Getenv("GLOW_TEST_TERMINAL_VIEWER")
	if viewer == "" {
		return
	}
	cmd := imageViewCommand{
		viewer:   viewer,
		source:   "test.png",
		rawInput: os.Getenv("GLOW_TEST_RAW_INPUT") == "1",
	}
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}
}
