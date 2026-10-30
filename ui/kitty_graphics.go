package ui

import (
	"bytes"
	"io"
	"os"
	"time"

	"github.com/muesli/cancelreader"
	"golang.org/x/term"
)

const kittyQueryID = "2147483646"

// Probe before Bubble Tea starts, so two readers never compete for terminal replies.
func supportsKittyGraphics() bool {
	if !term.IsTerminal(int(os.Stdout.Fd())) || os.Getenv("TERM") == "dumb" {
		return false
	}
	tty, state, err := openRawTerminal()
	if err != nil {
		return false
	}
	defer func() {
		_ = term.Restore(int(tty.Fd()), state)
		_ = tty.Close()
	}()
	reader, err := cancelreader.NewReader(tty)
	if err != nil {
		return false
	}
	defer func() { _ = reader.Close() }()
	query := "\x1b_Gi=" + kittyQueryID + ",a=q,t=d,f=24,s=1,v=1;AAAA\x1b\\"
	query = wrapKittyGraphics(query)
	if _, err := io.WriteString(tty, query); err != nil {
		return false
	}
	timer := time.AfterFunc(time.Second, func() { reader.Cancel() })
	defer timer.Stop()
	return readKittyGraphicsReply(reader)
}

func wrapKittyGraphics(sequence string) string {
	if os.Getenv("TMUX") != "" {
		return "\x1bPtmux;" + string(bytes.ReplaceAll([]byte(sequence), []byte("\x1b"), []byte("\x1b\x1b"))) + "\x1b\\"
	}
	return sequence
}

func readKittyGraphicsReply(reader io.Reader) bool {
	var response []byte
	buf := make([]byte, 128)
	prefix := []byte("\x1b_Gi=" + kittyQueryID + ";")
	for len(response) < 4096 {
		n, err := reader.Read(buf)
		response = append(response, buf[:n]...)
		if start := bytes.Index(response, prefix); start >= 0 {
			payload := response[start+len(prefix):]
			if end := bytes.Index(payload, []byte("\x1b\\")); end >= 0 {
				return string(payload[:end]) == "OK"
			}
		}
		if err != nil || n == 0 {
			return false
		}
	}
	return false
}
