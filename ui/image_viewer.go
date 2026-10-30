package ui

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"golang.org/x/term"
)

const maxImageBytes = 32 << 20

type imageViewerFinishedMsg struct{ err error }

func (m pagerModel) imageStatus(message string) (pagerModel, tea.Cmd) {
	cmd := m.showStatusMessage(pagerStatusMessage{message, false})
	return m, cmd
}

func (m pagerModel) beginImageSelection() (pagerModel, tea.Cmd) {
	message := ""
	switch {
	case m.common.cfg.ImageViewer == "":
		message = "Set imageViewer: kittyview in glow.yml to view images"
	case !m.common.cfg.KittyGraphics:
		message = "Cannot confirm Kitty graphics support in this terminal"
	}
	if message != "" {
		return m.imageStatus(message)
	}
	images := documentImages(m.currentDocument.Body)
	if len(images) == 0 {
		return m.imageStatus("No Markdown images in this document")
	}
	items := make([]list.Item, 0, len(images))
	for _, image := range images {
		items = append(items, image)
	}
	picker := list.New(items, list.NewDefaultDelegate(), m.common.width, m.common.height)
	picker.Title = "Document Images"
	picker.KeyMap.CursorUp.SetKeys("up", "k", "ctrl+p")
	picker.KeyMap.CursorDown.SetKeys("down", "j", "ctrl+n")
	m.imagePicker = &picker
	m.state = pagerStateImages
	return m, nil
}

func (m pagerModel) updateImageSelection(msg tea.Msg) (pagerModel, tea.Cmd) {
	if key, ok := msg.(tea.KeyPressMsg); ok {
		switch key.String() {
		case keyEsc:
			if m.imagePicker.FilterState() == list.Unfiltered {
				m.imagePicker = nil
				m.state = pagerStateBrowse
				return m, nil
			}
		case "q":
			if m.imagePicker.FilterState() != list.Filtering {
				m.imagePicker = nil
				m.state = pagerStateBrowse
				return m, nil
			}
		case keyEnter:
			if m.imagePicker.FilterState() != list.Filtering {
				return m.openSelectedImage()
			}
		}
	}
	var cmd tea.Cmd
	*m.imagePicker, cmd = m.imagePicker.Update(msg)
	return m, cmd
}

func (m pagerModel) openSelectedImage() (pagerModel, tea.Cmd) {
	image, ok := m.imagePicker.SelectedItem().(documentImage)
	if !ok {
		return m, nil
	}
	documentURL := ""
	if m.currentDocument.localPath == "" {
		documentURL = m.common.cfg.SourceURL
	}
	source, err := resolveImageSource(image.source, m.currentDocument.localPath, documentURL)
	m.imagePicker = nil
	m.state = pagerStateBrowse
	if err != nil {
		return m.imageStatus(err.Error())
	}
	if _, err := exec.LookPath(m.common.cfg.ImageViewer); err != nil {
		return m.imageStatus("Image viewer executable not found")
	}
	m.viewingImage = true
	return m, tea.Exec(&imageViewCommand{viewer: m.common.cfg.ImageViewer, source: source, rawInput: m.common.cfg.rawInput}, func(err error) tea.Msg {
		return imageViewerFinishedMsg{err: err}
	})
}

// imageViewCommand owns terminal I/O while Bubble Tea is paused.
type imageViewCommand struct {
	viewer   string
	source   string
	rawInput bool
}

func (*imageViewCommand) SetStdin(io.Reader)  {}
func (*imageViewCommand) SetStdout(io.Writer) {}
func (*imageViewCommand) SetStderr(io.Writer) {}

func (c *imageViewCommand) Run() error {
	tty, state, err := openRawTerminal()
	if err != nil {
		return fmt.Errorf("cannot open image viewer terminal: %w", err)
	}
	defer func() {
		// WithoutRenderer does not restore raw mode before restarting input.
		if c.rawInput {
			_, _ = term.MakeRaw(int(tty.Fd()))
		} else {
			_ = term.Restore(int(tty.Fd()), state)
		}
		_ = tty.Close()
	}()
	_, _ = io.WriteString(tty, "\x1b[?1049h\x1b[2J\x1b[H\x1b[?25h")
	defer func() {
		_, _ = io.WriteString(tty, wrapKittyGraphics("\x1b_Ga=d,d=a\x1b\\")+"\x1b[?1049l")
	}()
	path := c.source
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		_, _ = io.WriteString(tty, "Downloading image...\r\n")
		path, err = downloadImage(context.Background(), path)
		if err != nil {
			return err
		}
		defer func() { _ = os.Remove(path) }()
	}
	cmd := exec.CommandContext(context.Background(), c.viewer, "--", path) //nolint:gosec // Executable comes from user config; image is a separate argument.
	cmd.Stdin, cmd.Stdout, cmd.Stderr = tty, tty, io.Discard
	// Kittyview's placeholder rows use LF and require normal terminal output
	// processing to return to column zero, just like invocation from a shell.
	if err := term.Restore(int(tty.Fd()), state); err != nil {
		return fmt.Errorf("cannot restore image viewer terminal: %w", err)
	}
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("image viewer failed: %w", err)
	}
	if _, err := term.MakeRaw(int(tty.Fd())); err != nil {
		return fmt.Errorf("cannot read image viewer input: %w", err)
	}
	_, _ = io.WriteString(tty, "\r\nPress Enter or Esc to return to Glow.\r\n")
	// Decode complete input events so an escape in a terminal reply is not
	// mistaken for the Escape key. No renderer may overwrite the image.
	_, err = tea.NewProgram(imageReturnModel{}, tea.WithInput(tty), tea.WithOutput(tty), tea.WithoutRenderer()).Run()
	if err != nil {
		return fmt.Errorf("cannot wait for image viewer dismissal: %w", err)
	}
	return nil
}

type imageReturnModel struct{}

func (imageReturnModel) Init() tea.Cmd  { return nil }
func (imageReturnModel) View() tea.View { return tea.NewView("") }
func (m imageReturnModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyPressMsg); ok {
		switch key.String() {
		case keyEnter, keyCtrlJ, keyEsc, keyCtrlC:
			return m, tea.Quit
		}
	}
	return m, nil
}

func downloadImage(ctx context.Context, source string) (path string, err error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, source, nil)
	if err != nil {
		return "", fmt.Errorf("invalid image request: %w", err)
	}
	client := &http.Client{CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 5 || (req.URL.Scheme != "http" && req.URL.Scheme != "https") {
			return fmt.Errorf("image redirect is not allowed")
		}
		return nil
	}}
	response, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("cannot download image: %w", err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("image download returned HTTP %d", response.StatusCode)
	}
	if response.ContentLength > maxImageBytes {
		return "", fmt.Errorf("image exceeds 32 MiB download limit")
	}
	file, err := os.CreateTemp("", "glow-image-*")
	if err != nil {
		return "", fmt.Errorf("cannot create temporary image: %w", err)
	}
	path = file.Name()
	defer func() {
		_ = file.Close()
		if err != nil {
			_ = os.Remove(path)
		}
	}()
	n, err := io.Copy(file, io.LimitReader(response.Body, maxImageBytes+1))
	if err != nil {
		return path, fmt.Errorf("cannot save image: %w", err)
	}
	if n > maxImageBytes {
		return path, fmt.Errorf("image exceeds 32 MiB download limit")
	}
	if err := file.Close(); err != nil {
		return path, fmt.Errorf("cannot close temporary image: %w", err)
	}
	return path, nil
}
