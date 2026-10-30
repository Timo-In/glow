package ui

import (
	"fmt"
	"html"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

type documentImage struct {
	label  string
	source string
}

func (i documentImage) Title() string       { return i.label }
func (i documentImage) Description() string { return i.source }
func (i documentImage) FilterValue() string { return i.label + " " + i.source }

func documentImages(body string) []documentImage {
	source := []byte(body)
	doc := goldmark.New().Parser().Parse(text.NewReader(source))
	var images []documentImage
	seen := make(map[string]bool)
	_ = ast.Walk(doc, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		image, ok := node.(*ast.Image)
		if !entering || !ok {
			return ast.WalkContinue, nil
		}
		destination := html.UnescapeString(string(util.UnescapePunctuations(image.Destination)))
		if destination == "" || seen[destination] {
			return ast.WalkContinue, nil
		}
		seen[destination] = true
		label := imageLabel(image, source)
		if strings.TrimSpace(label) == "" {
			label = destination
		}
		images = append(images, documentImage{label: label, source: destination})
		return ast.WalkContinue, nil
	})
	return images
}

func imageLabel(image *ast.Image, source []byte) string {
	var label strings.Builder
	_ = ast.Walk(image, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch value := node.(type) {
		case *ast.Text:
			label.Write(value.Value(source))
			if value.SoftLineBreak() {
				label.WriteByte('\n')
			}
		case *ast.String:
			label.Write(value.Value)
		}
		return ast.WalkContinue, nil
	})
	return label.String()
}

func resolveImageSource(source, documentPath, documentURL string) (string, error) {
	u, err := url.Parse(source)
	if err != nil {
		return "", fmt.Errorf("invalid image location: %w", err)
	}
	if documentURL != "" {
		base, err := url.Parse(documentURL)
		if err != nil {
			return "", fmt.Errorf("invalid document URL: %w", err)
		}
		u = base.ResolveReference(u)
	}
	if u.Scheme != "" || u.Host != "" {
		if (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			return "", fmt.Errorf("image URL must use HTTP or HTTPS")
		}
		u.Fragment = ""
		return u.String(), nil
	}
	if u.Path == "" {
		return "", fmt.Errorf("image path is empty")
	}
	path := filepath.FromSlash(u.Path)
	if !filepath.IsAbs(path) {
		path = filepath.Join(filepath.Dir(documentPath), path)
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("cannot resolve image path: %w", err)
	}
	return absolute, nil
}
