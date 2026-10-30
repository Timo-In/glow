package ui

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
)

func TestDownloadImage(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("TMPDIR", dir)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/redirect":
			http.Redirect(w, r, "/image", http.StatusFound)
		case "/image":
			_, _ = w.Write([]byte("image payload"))
		case "/large":
			w.Header().Set("Content-Length", "33554433")
		case "/stream-large":
			w.(http.Flusher).Flush()
			_, _ = io.CopyN(w, &zeroReader{}, maxImageBytes+1)
		case "/truncated":
			w.Header().Set("Content-Length", "100")
			_, _ = w.Write([]byte("short"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	for _, route := range []string{"/image", "/redirect"} {
		path, err := downloadImage(context.Background(), server.URL+route)
		if err != nil {
			t.Fatal(err)
		}
		data, err := os.ReadFile(path)
		if err != nil || string(data) != "image payload" {
			t.Fatalf("unexpected download: %q, %v", data, err)
		}
		_ = os.Remove(path)
	}
	for _, route := range []string{"/missing", "/large", "/stream-large", "/truncated"} {
		if _, err := downloadImage(context.Background(), server.URL+route); err == nil {
			t.Fatalf("expected failure for %s", route)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := downloadImage(ctx, server.URL+"/image"); err == nil {
		t.Fatal("canceled request succeeded")
	}
	files, err := os.ReadDir(dir)
	if err != nil || len(files) != 0 {
		t.Fatalf("failed downloads left temporary files: %v, %v", files, err)
	}
}

func TestDownloadHTTPSImage(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("secure image"))
	}))
	defer server.Close()
	previous := http.DefaultTransport
	http.DefaultTransport = server.Client().Transport
	t.Cleanup(func() { http.DefaultTransport = previous })
	path, err := downloadImage(context.Background(), server.URL+"/image.png")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(path) }()
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "secure image" {
		t.Fatalf("unexpected HTTPS download: %q, %v", data, err)
	}
}

type zeroReader struct{}

func (*zeroReader) Read(p []byte) (int, error) {
	clear(p)
	return len(p), nil
}

func imageTestModel(t *testing.T, cfg Config) model {
	t.Helper()
	m := newModel(cfg, strings.Repeat("paragraph\n\n", 50)+"![Photo](photo.png)\n").(model)
	m.common.width, m.common.height = 80, 24
	m.pager.setSize(80, 24)
	m.pager.setContent(strings.Repeat("line\n", 100))
	m.pager.viewport.SetYOffset(15)
	t.Cleanup(func() {
		if m.pager.watcher != nil {
			_ = m.pager.watcher.Close()
		}
	})
	return m
}

func TestImageShortcutChecksConfigurationAndCapability(t *testing.T) {
	for _, tc := range []struct {
		name string
		cfg  Config
		want string
	}{
		{"disabled", Config{}, "Set imageViewer"},
		{"unsupported", Config{ImageViewer: "kittyview"}, "Kitty graphics support"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := imageTestModel(t, tc.cfg)
			updated, _ := m.Update(tea.KeyPressMsg{Code: 'o', Mod: tea.ModCtrl})
			got := updated.(model)
			if got.pager.imagePicker != nil || got.pager.viewingImage || !strings.Contains(got.pager.statusMessage, tc.want) {
				t.Fatalf("unexpected image state: %#v", got.pager)
			}
			if got.pager.statusMessageTimer != nil {
				got.pager.statusMessageTimer.Stop()
			}
		})
	}
}

func TestImagePickerReturnsToDocumentAndPreservesPosition(t *testing.T) {
	m := imageTestModel(t, Config{ImageViewer: "kittyview", KittyGraphics: true})
	updated, _ := m.Update(tea.KeyPressMsg{Code: 'o', Mod: tea.ModCtrl})
	m = updated.(model)
	if m.pager.imagePicker == nil || m.pager.state != pagerStateImages {
		t.Fatal("image picker did not open")
	}
	updated, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	m = updated.(model)
	if m.state != stateShowDocument || m.pager.state != pagerStateBrowse || m.pager.viewport.YOffset() != 15 {
		t.Fatal("cancel did not return to original document position")
	}
	m.pager.viewingImage = true
	updated, _ = m.Update(imageViewerFinishedMsg{})
	m = updated.(model)
	if m.pager.viewingImage || m.pager.viewport.YOffset() != 15 {
		t.Fatal("viewer completion changed document position")
	}
}

func TestViewerResumeSizeReportPreservesSearchPosition(t *testing.T) {
	m := imageTestModel(t, Config{})
	m.pager.searchQuery = "line"
	updated, cmd := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(model)
	if cmd != nil || m.pager.viewport.YOffset() != 15 {
		t.Fatal("unchanged terminal dimensions triggered a rerender after image viewing")
	}
}

func TestRemoteDocumentRetainsSourceURL(t *testing.T) {
	sourceURL := "https://example.com/docs/readme.md"
	m := newModel(Config{SourceURL: sourceURL}, "![Photo](images/a.png)").(model)
	if m.pager.currentDocument.Note != sourceURL || m.pager.currentDocument.localPath != "" {
		t.Fatal("remote document source was lost")
	}
}

func TestImagePickerFiltersAndSearchHasPriority(t *testing.T) {
	m := imageTestModel(t, Config{ImageViewer: "kittyview", KittyGraphics: true})
	updated, _ := m.Update(tea.KeyPressMsg{Code: '/', Text: "/"})
	m = updated.(model)
	updated, _ = m.Update(tea.KeyPressMsg{Code: 'o', Mod: tea.ModCtrl})
	m = updated.(model)
	if m.pager.imagePicker != nil || m.pager.state != pagerStateSearch {
		t.Fatal("image shortcut intercepted search input")
	}
	m.pager.state = pagerStateBrowse
	updated, _ = m.Update(tea.KeyPressMsg{Code: 'o', Mod: tea.ModCtrl})
	m = updated.(model)
	updated, _ = m.Update(tea.KeyPressMsg{Code: '/', Text: "/"})
	m = updated.(model)
	updated, _ = m.Update(tea.KeyPressMsg{Code: 'q', Text: "q"})
	m = updated.(model)
	if m.pager.imagePicker.FilterState() != list.Filtering || m.state != stateShowDocument {
		t.Fatal("global shortcut interrupted image filtering")
	}
}

func TestImageViewingSuspendsRawPainter(t *testing.T) {
	var output bytes.Buffer
	painter := &rawPainter{w: &output}
	stale := painter.paint("old frame")
	painter.setImageViewing(true)
	stale()
	painter.paint("must not overwrite image")()
	if strings.Contains(output.String(), "frame") || strings.Contains(output.String(), "overwrite") {
		t.Fatal("painter wrote while image viewer owned terminal")
	}
	painter.setImageViewing(false)
	painter.paint("restored document")()
	if !strings.Contains(output.String(), "restored document") {
		t.Fatal("painter did not resume")
	}
}
