package tui

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"sync"

	tea "github.com/charmbracelet/bubbletea"

	"scdl/internal/app"
)

func runDownloadCmd(ctx context.Context, url, outputPath string, force bool, logCh chan string, statusCh chan string, confirmCh chan string, confirmRespCh chan bool) tea.Cmd {
	return func() tea.Msg {
		var buf bytes.Buffer
		writer := newLineWriter(io.MultiWriter(&buf), logCh)
		opts := app.Options{
			OutputDir: strings.TrimSpace(outputPath),
			Force:     force,
			Status: func(message string) {
				select {
				case statusCh <- message:
				default:
				}
			},
			ConfirmOverwrite: func(path string) (bool, error) {
				if confirmCh == nil || confirmRespCh == nil {
					return false, fmt.Errorf("overwrite confirmation unavailable")
				}
				select {
				case confirmCh <- path:
				default:
				}
				select {
				case response := <-confirmRespCh:
					return response, nil
				case <-ctx.Done():
					return false, ctx.Err()
				}
			},
		}
		err := app.Run(ctx, url, opts, strings.NewReader(""), writer)
		writer.Flush()
		close(logCh)
		close(statusCh)
		close(confirmCh)
		close(confirmRespCh)
		return runCompleteMsg{log: buf.String(), err: err}
	}
}

func appendLog(existing, line string, limit int) string {
	if line == "" {
		return existing
	}
	if existing == "" {
		return line
	}
	combined := existing + "\n" + line
	lines := strings.Split(combined, "\n")
	if len(lines) <= limit {
		return combined
	}
	return strings.Join(lines[len(lines)-limit:], "\n")
}

func tailLines(value string, limit int) string {
	lines := strings.Split(strings.TrimSpace(value), "\n")
	if len(lines) <= limit {
		return strings.Join(lines, "\n")
	}
	return strings.Join(lines[len(lines)-limit:], "\n")
}

func listenLogs(ch <-chan string) tea.Cmd {
	return func() tea.Msg {
		if ch == nil {
			return nil
		}
		line, ok := <-ch
		if !ok {
			return nil
		}
		return logMsg(line)
	}
}

func listenStatus(ch <-chan string) tea.Cmd {
	return func() tea.Msg {
		if ch == nil {
			return nil
		}
		line, ok := <-ch
		if !ok {
			return nil
		}
		return statusMsg(line)
	}
}

func listenConfirm(ch <-chan string) tea.Cmd {
	return func() tea.Msg {
		if ch == nil {
			return nil
		}
		path, ok := <-ch
		if !ok {
			return nil
		}
		return confirmMsg(path)
	}
}

func statusPercent(status string) float64 {
	steps := []string{
		"Fetching page data",
		"Parsing metadata",
		"Preparing output",
		"Downloading image",
		"Downloading audio",
		"Creating video",
		"Done",
	}
	for i, step := range steps {
		if status == step {
			return float64(i+1) / float64(len(steps))
		}
	}
	return 0
}

type lineWriter struct {
	mu  sync.Mutex
	buf bytes.Buffer
	out io.Writer
	ch  chan<- string
}

func newLineWriter(out io.Writer, ch chan<- string) *lineWriter {
	return &lineWriter{out: out, ch: ch}
}

func (w *lineWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	n, err := w.out.Write(p)
	if n > 0 && w.ch != nil {
		w.buf.Write(p[:n])
		w.flushLines()
	}
	return n, err
}

func (w *lineWriter) Flush() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.ch == nil {
		return
	}
	if w.buf.Len() > 0 {
		select {
		case w.ch <- strings.TrimRight(w.buf.String(), "\n"):
		default:
		}
		w.buf.Reset()
	}
}

func (w *lineWriter) flushLines() {
	for {
		line, err := w.buf.ReadString('\n')
		if err != nil {
			w.buf.WriteString(line)
			return
		}
		select {
		case w.ch <- strings.TrimRight(line, "\n"):
		default:
		}
	}
}
