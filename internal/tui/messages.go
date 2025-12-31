package tui

type runCompleteMsg struct {
	log string
	err error
}

type logMsg string
type statusMsg string
type confirmMsg string
