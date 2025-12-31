package tui

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

type Config struct {
	URL       string
	OutputDir string
	Force     bool
}

type focusTarget int

const (
	focusURL focusTarget = iota
	focusOutput
	focusLogs
)

type model struct {
	ctx context.Context

	urlInput    textinput.Model
	outputInput textinput.Model
	focusIndex  int
	force       bool
	defaultOut  string

	running     bool
	done        bool
	log         string
	err         error
	showLogs    bool
	confirming  bool
	confirmPath string

	spinner    spinner.Model
	progress   progress.Model
	status     string
	termWidth  int
	termHeight int

	logViewport viewport.Model
	logFocused  bool
	autoFollow  bool

	logCh         chan string
	statusCh      chan string
	confirmCh     chan string
	confirmRespCh chan bool
}

func newModel(ctx context.Context, cfg Config) model {
	urlInput := textinput.New()
	urlInput.Placeholder = "https://soundcloud.com/artist/track"
	urlInput.SetValue(cfg.URL)
	urlInput.Focus()
	urlInput.CharLimit = 0
	urlInput.Width = 60

	outputInput := textinput.New()
	outputInput.Placeholder = fmt.Sprintf("Default: %s", defaultOutputPlaceholder(cfg.OutputDir))
	outputInput.CharLimit = 0
	outputInput.Width = 60

	sp := spinner.New()
	sp.Spinner = spinner.Dot

	prog := progress.New(progress.WithDefaultGradient())

	vp := viewport.New(60, 8)
	vp.SetContent("")

	return model{
		ctx:         ctx,
		urlInput:    urlInput,
		outputInput: outputInput,
		focusIndex:  0,
		force:       cfg.Force,
		defaultOut:  cfg.OutputDir,
		spinner:     sp,
		progress:    prog,
		status:      "Idle",
		logViewport: vp,
		autoFollow:  true,
	}
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		updated, cmd, handled := m.handleKey(msg)
		if handled {
			return updated, cmd
		}
		m = updated
	case spinner.TickMsg:
		if m.running {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
	case logMsg:
		m.log = appendLog(m.log, string(msg), 200)
		m.logViewport.SetContent(m.log)
		if m.autoFollow {
			m.logViewport.GotoBottom()
		}
		return m, listenLogs(m.logCh)
	case statusMsg:
		m.status = string(msg)
		cmd := m.progress.SetPercent(statusPercent(m.status))
		return m, tea.Batch(cmd, listenStatus(m.statusCh))
	case confirmMsg:
		m.confirming = true
		m.confirmPath = string(msg)
		return m, listenConfirm(m.confirmCh)
	case runCompleteMsg:
		m.running = false
		m.done = true
		m.log = msg.log
		m.err = msg.err
		m.logViewport.SetContent(m.log)
		if m.err == nil {
			m.urlInput.SetValue("")
			m.focusIndex = 0
			m.updateFocus()
		}
		return m, nil
	case tea.WindowSizeMsg:
		m.termWidth = msg.Width
		m.termHeight = msg.Height
		m.progress.Width = progressWidth(msg.Width)
		m.setViewportSize()
	case progress.FrameMsg:
		updated, cmd := m.progress.Update(msg)
		if progressModel, ok := updated.(progress.Model); ok {
			m.progress = progressModel
		}
		return m, cmd
	}

	if m.logFocused {
		var cmd tea.Cmd
		m.logViewport, cmd = m.logViewport.Update(msg)
		m.autoFollow = m.logViewport.AtBottom()
		return m, cmd
	}

	if m.confirming || m.running {
		return m, nil
	}

	var cmd tea.Cmd
	switch m.focusIndex {
	case 0:
		m.urlInput, cmd = m.urlInput.Update(msg)
	case 1:
		m.outputInput, cmd = m.outputInput.Update(msg)
	}
	return m, cmd
}

func (m model) handleKey(msg tea.KeyMsg) (model, tea.Cmd, bool) {
	key := msg.String()
	if m.confirming {
		switch key {
		case keyYes:
			m.sendConfirmResponse(true)
		case keyNo, keyEsc:
			m.sendConfirmResponse(false)
		case keyQuit:
			m.sendConfirmResponse(false)
			return m, tea.Quit, true
		}
		return m, nil, true
	}

	if m.running {
		switch key {
		case keyQuit:
			return m, tea.Quit, true
		case keyToggleLogs:
			m.toggleLogs()
			return m, nil, true
		case keyTab, keyShiftTab:
			m.advanceFocus(key)
			return m, nil, true
		case keyUp, keyDown, keyPgUp, keyPgDown:
			if m.logFocused {
				return m, nil, false
			}
			return m, nil, true
		default:
			return m, nil, true
		}
	}

	switch key {
	case keyQuit:
		return m, tea.Quit, true
	case keyTab, keyShiftTab:
		m.advanceFocus(key)
		return m, nil, true
	case keyToggleLogs:
		m.toggleLogs()
		return m, nil, true
	case keyEnter:
		m.err = nil
		m.log = ""
		m.done = false
		m.logViewport.SetContent("")
		m.autoFollow = true
		return m, m.startRun(), true
	case keyUp, keyDown, keyPgUp, keyPgDown:
		return m, nil, true
	}

	return m, nil, false
}

func (m *model) startRun() tea.Cmd {
	url := strings.TrimSpace(m.urlInput.Value())
	if url == "" {
		m.err = fmt.Errorf("soundcloud URL is required")
		return nil
	}
	outputDir := strings.TrimSpace(m.outputInput.Value())
	if outputDir == "" {
		outputDir = m.defaultOut
	}

	m.running = true
	m.logCh = make(chan string, 32)
	m.statusCh = make(chan string, 16)
	m.confirmCh = make(chan string, 1)
	m.confirmRespCh = make(chan bool, 1)
	return tea.Batch(
		m.spinner.Tick,
		runDownloadCmd(m.ctx, url, outputDir, m.force, m.logCh, m.statusCh, m.confirmCh, m.confirmRespCh),
		listenLogs(m.logCh),
		listenStatus(m.statusCh),
		listenConfirm(m.confirmCh),
	)
}

func defaultOutputPlaceholder(configured string) string {
	if strings.TrimSpace(configured) != "" {
		return configured
	}
	if cwd, err := os.Getwd(); err == nil {
		return cwd
	}
	return "."
}

func (m *model) updateFocus() {
	inputs := []*textinput.Model{&m.urlInput, &m.outputInput}
	for i, input := range inputs {
		if i == m.focusIndex {
			input.Focus()
		} else {
			input.Blur()
		}
	}
}

func (m *model) setViewportSize() {
	if m.termWidth <= 0 {
		m.logViewport.Width = 60
		m.logViewport.Height = 8
		return
	}
	m.logViewport.Width = maxInt(20, m.termWidth-4)
	m.logViewport.Height = 8
}

func (m *model) advanceFocus(key string) {
	if key == keyShiftTab {
		m.focusIndex--
	} else {
		m.focusIndex++
	}
	maxFocus := 1
	if m.showLogs {
		maxFocus = 2
	}
	if m.focusIndex > maxFocus {
		m.focusIndex = 0
	}
	if m.focusIndex < 0 {
		m.focusIndex = maxFocus
	}
	m.logFocused = m.showLogs && m.focusIndex == 2
	m.updateFocus()
}

func (m *model) toggleLogs() {
	m.showLogs = !m.showLogs
	if !m.showLogs {
		m.logFocused = false
		if m.focusIndex > 1 {
			m.focusIndex = 1
		}
		m.updateFocus()
	}
}

func (m *model) sendConfirmResponse(response bool) {
	if m.confirmRespCh != nil {
		select {
		case m.confirmRespCh <- response:
		default:
		}
	}
	m.confirming = false
	m.confirmPath = ""
}

func (m model) logView(height int) string {
	if height <= 0 {
		return ""
	}
	if m.log == "" {
		return "(no logs yet)"
	}
	if m.logFocused {
		vp := m.logViewport
		vp.Height = height
		return vp.View()
	}
	return tailLines(m.log, height)
}
