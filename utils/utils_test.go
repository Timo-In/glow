package utils

import (
	"strings"
	"testing"

	"charm.land/glamour/v2"
	"charm.land/glamour/v2/styles"
)

func TestDynamicHorizontalRule(t *testing.T) {
	tests := []struct {
		name       string
		width      int
		wantDashes int
	}{
		{
			name:       "width 80",
			width:      80,
			wantDashes: 76, // 80 - margin*2 (4)
		},
		{
			name:       "width 60",
			width:      60,
			wantDashes: 56, // 60 - margin*2 (4)
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r, err := glamour.NewTermRenderer(
				GlamourStyle(styles.DarkStyle, false, tc.width),
				glamour.WithWordWrap(tc.width),
			)
			if err != nil {
				t.Fatalf("unexpected error creating renderer: %v", err)
			}

			out, err := r.Render("---\n")
			if err != nil {
				t.Fatalf("unexpected error rendering: %v", err)
			}

			expectedLine := strings.Repeat("─", tc.wantDashes)
			if !strings.Contains(out, expectedLine) {
				t.Errorf("expected rendered output to contain %d full-width dashes %q, got %q", tc.wantDashes, expectedLine, out)
			}

			if strings.Contains(out, "--------") {
				t.Errorf("expected default dashes '--------' to be replaced, got %q", out)
			}
		})
	}
}
