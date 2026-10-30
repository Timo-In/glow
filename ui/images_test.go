package ui

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestDocumentImages(t *testing.T) {
	body := "![Local](<images/my photo.png>)\n![Remote](https://example.com/a.png?a=1&amp;b=2)\n" +
		"![Reference][diagram]\n[diagram]: ../diagram.svg\n" +
		"![Duplicate](../diagram.svg)\n`![Inline code](no.png)`\n```md\n![Code block](no.png)\n```\n"
	images := documentImages(body)
	if len(images) != 3 {
		t.Fatalf("expected 3 images, got %#v", images)
	}
	if images[0].source != "images/my photo.png" || images[0].label != "Local" {
		t.Fatalf("unexpected local image: %#v", images[0])
	}
	if images[1].source != "https://example.com/a.png?a=1&b=2" {
		t.Errorf("URL entities not decoded: %q", images[1].source)
	}
	if images[2].source != "../diagram.svg" {
		t.Errorf("reference not resolved: %#v", images[2])
	}
}

func TestResolveImageSource(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "notes", "readme.md")
	tests := []struct {
		name, source, baseURL, want string
		wantError                   bool
	}{
		{name: "relative local", source: "../images/a%20b.png", want: filepath.Join(dir, "images", "a b.png")},
		{name: "absolute local", source: filepath.Join(dir, "a.png"), want: filepath.Join(dir, "a.png")},
		{name: "https", source: "https://example.com/a.png?q=1#fragment", want: "https://example.com/a.png?q=1"},
		{name: "remote relative", source: "../img/a.png", baseURL: "https://example.com/docs/readme.md", want: "https://example.com/img/a.png"},
		{name: "remote root", source: "/a.png", baseURL: "https://example.com/docs/readme.md", want: "https://example.com/a.png"},
		{name: "protocol relative", source: "//cdn.example.com/a.png", baseURL: "https://example.com/docs/readme.md", want: "https://cdn.example.com/a.png"},
		{name: "unsupported scheme", source: "javascript:alert(1)", wantError: true},
		{name: "file URL", source: "file:///etc/passwd", wantError: true},
		{name: "data URL", source: "data:image/png;base64,AAAA", wantError: true},
		{name: "empty", source: "", wantError: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := resolveImageSource(tc.source, file, tc.baseURL)
			if tc.wantError {
				if err == nil {
					t.Fatalf("expected rejection, got %q", got)
				}
				return
			}
			if err != nil || got != tc.want {
				t.Fatalf("got %q, %v; want %q", got, err, tc.want)
			}
		})
	}
}

func TestReadKittyGraphicsReply(t *testing.T) {
	for _, tc := range []struct {
		name, reply string
		want        bool
	}{
		{"supported", "\x1b_Gi=" + kittyQueryID + ";OK\x1b\\", true},
		{"unrelated reply", "\x1b_Gi=31;OK\x1b\\", false},
		{"refused", "\x1b_Gi=" + kittyQueryID + ";ENOTSUPPORTED\x1b\\", false},
		{"no reply", "", false},
		{"incomplete", "\x1b_Gi=" + kittyQueryID + ";OK", false},
		{"bounded", strings.Repeat("x", 5000), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := readKittyGraphicsReply(strings.NewReader(tc.reply)); got != tc.want {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}
