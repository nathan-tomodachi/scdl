package tui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

func (m model) View() string {
	title := lipgloss.NewStyle().Bold(true).Render("SoundCloud Downloader")
	label := lipgloss.NewStyle().Foreground(lipgloss.Color("68")).Render
	help := lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render

	header := lipgloss.JoinVertical(lipgloss.Left,
		title,
		"",
		label("URL"),
		m.urlInput.View(),
		"",
		label("Output directory (optional)"),
		m.outputInput.View(),
		"",
		help("Tab to switch fields. Enter to start. Ctrl+l to toggle logs. Ctrl+c to quit."),
	)

	status := ""
	if m.running {
		progressView := m.progress
		progressView.Width = progressWidth(m.termWidth)
		statusLine := m.status
		if statusLine == "" {
			statusLine = "Working"
		}
		status = lipgloss.JoinVertical(lipgloss.Left,
			m.spinner.View()+" "+statusLine,
			progressView.View(),
		)
	} else if m.done && m.err == nil {
		doneStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true)
		status = doneStyle.Render("Done.")
	}

	errors := ""
	if m.err != nil {
		errors = "Error: " + m.err.Error()
	}

	confirm := ""
	if m.confirming {
		confirm = renderConfirmModal(m.confirmPath)
	}

	sections := []string{header}
	if status != "" {
		sections = append(sections, "", status)
	}
	if confirm != "" {
		sections = append(sections, "", confirm)
	}
	if errors != "" {
		sections = append(sections, "", errors)
	}
	baseHeight := lipgloss.Height(lipgloss.JoinVertical(lipgloss.Left, sections...))

	if m.showLogs {
		logHeight := logViewportHeight(m.termHeight, baseHeight)
		if logHeight > 0 {
			logs := lipgloss.JoinVertical(lipgloss.Left, "Log:", m.logView(logHeight))
			sections = append(sections, "", logs)
		}
	}

	content := lipgloss.JoinVertical(lipgloss.Left, sections...)
	if m.termHeight > 0 {
		return lipgloss.NewStyle().Height(m.termHeight).Render(content)
	}
	return content
}

func renderConfirmModal(path string) string {
	title := lipgloss.NewStyle().Bold(true).Render("Overwrite file?")
	body := fmt.Sprintf("%s\n%s\n[y]es / [n]o", title, path)
	width := minInt(72, maxInt(24, len([]rune(path))+6))
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(1, 2).
		Width(width)
	return style.Render(body)
}

func progressWidth(termWidth int) int {
	if termWidth <= 0 {
		return 36
	}
	width := termWidth - 6
	if width > 60 {
		return 60
	}
	if width < 20 {
		return 20
	}
	return width
}

func logViewportHeight(termHeight, baseHeight int) int {
	if termHeight <= 0 {
		return 8
	}
	maxHeight := minInt(12, maxInt(3, termHeight/3))
	available := termHeight - baseHeight - 1
	if available <= 0 {
		return 0
	}
	if available < maxHeight {
		return available
	}
	return maxHeight
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
