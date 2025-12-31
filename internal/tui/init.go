package tui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/viper"

	"scdl/internal/setup"
)

type InitConfig struct {
	ConfigPath string
}

type initStep int

const (
	stepChecking initStep = iota
	stepInstallPrompt
	stepInstalling
	stepOutputPrompt
	stepSaving
	stepDone
	stepError
)

type initMsg struct {
	status setup.Status
	err    error
}

type initModel struct {
	ctx        context.Context
	step       initStep
	spinner    spinner.Model
	statusText string
	err        error

	missing    []setup.Dependency
	installer  setup.Installer
	configPath string

	installChoice bool
	output        textinput.Model
}

func RunInit(ctx context.Context, cfg InitConfig) error {
	m := newInitModel(ctx, cfg)
	prog := tea.NewProgram(m, tea.WithAltScreen())
	_, err := prog.Run()
	return err
}

func newInitModel(ctx context.Context, cfg InitConfig) initModel {
	sp := spinner.New()
	sp.Spinner = spinner.Dot

	out := textinput.New()
	out.Placeholder = fmt.Sprintf("Default: %s", resolveDefaultOutput(cfg.ConfigPath))
	out.CharLimit = 0
	out.Width = 60

	return initModel{
		ctx:           ctx,
		step:          stepChecking,
		spinner:       sp,
		statusText:    "Checking dependencies",
		configPath:    cfg.ConfigPath,
		installChoice: true,
		output:        out,
	}
}

func (m initModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, checkDepsCmd(m.ctx))
}

func (m initModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case initMsg:
		if msg.err != nil {
			m.step = stepError
			m.err = msg.err
			return m, nil
		}
		if m.step == stepSaving {
			m.step = stepDone
			m.statusText = "Setup complete"
			return m, nil
		}
		if m.step == stepInstalling {
			status := msg.status
			if len(status.Missing) > 0 {
				m.step = stepError
				m.err = fmt.Errorf("dependencies still missing: %s", setup.MissingSummary(status))
				return m, nil
			}
			m.step = stepOutputPrompt
			m.statusText = "Dependencies installed"
			m.output.Focus()
			return m, nil
		}
		status := msg.status
		if len(status.Missing) == 0 {
			m.step = stepOutputPrompt
			m.statusText = "All dependencies available"
			m.output.Focus()
			return m, nil
		}
		installer, err := setup.DetectInstaller()
		if err != nil {
			m.step = stepError
			m.err = fmt.Errorf("%s. Missing: %s", err, setup.MissingSummary(status))
			return m, nil
		}
		m.installer = installer
		m.missing = status.Missing
		m.step = stepInstallPrompt
		m.statusText = setup.MissingSummary(status)
		return m, nil
	}
	return m, nil
}

func (m initModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	if key == keyQuit {
		return m, tea.Quit
	}

	switch m.step {
	case stepInstallPrompt:
		switch key {
		case keyYes:
			m.installChoice = true
			m.step = stepInstalling
			m.statusText = "Installing dependencies"
			return m, installDepsCmd(m.ctx, m.installer, m.missing)
		case keyNo:
			m.installChoice = false
			m.step = stepOutputPrompt
			m.output.Focus()
			return m, nil
		case keyEnter:
			if m.installChoice {
				m.step = stepInstalling
				m.statusText = "Installing dependencies"
				return m, installDepsCmd(m.ctx, m.installer, m.missing)
			}
			m.step = stepOutputPrompt
			m.output.Focus()
			return m, nil
		case keyLeft, keyRight, keyUp, keyDown, keySpace:
			m.installChoice = !m.installChoice
			return m, nil
		case keyEsc:
			return m, tea.Quit
		}
	case stepOutputPrompt:
		if key == keyEnter {
			m.step = stepSaving
			m.statusText = "Saving configuration"
			return m, saveConfigCmd(m.ctx, m.output.Value(), m.configPath)
		}
		var cmd tea.Cmd
		m.output, cmd = m.output.Update(msg)
		return m, cmd
	case stepDone:
		if key == keyEnter {
			return m, tea.Quit
		}
	case stepError:
		if key == keyEnter {
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m initModel) View() string {
	header := "Init setup"
	status := m.statusText
	if m.step == stepChecking || m.step == stepInstalling || m.step == stepSaving {
		status = m.spinner.View() + " " + m.statusText
	}

	var body strings.Builder
	body.WriteString(header)
	body.WriteString("\n\n")
	body.WriteString(status)

	switch m.step {
	case stepInstallPrompt:
		body.WriteString("\n\n")
		body.WriteString("Install missing dependencies now?\n")
		body.WriteString(renderInstallChoice(m.installChoice))
		body.WriteString("\nUse left/right or up/down to toggle, Enter to continue.")
	case stepOutputPrompt:
		body.WriteString("\n\n")
		body.WriteString("Set default output directory (leave blank to skip):\n")
		body.WriteString(m.output.View())
	case stepDone:
		body.WriteString("\n\n")
		body.WriteString("Setup complete. Press Enter to exit.")
	case stepError:
		body.WriteString("\n\n")
		body.WriteString("Error: ")
		body.WriteString(m.err.Error())
		body.WriteString("\nPress Enter to exit.")
	}

	return body.String()
}

func renderInstallChoice(install bool) string {
	yes := "[ ] yes"
	no := "[ ] no"
	if install {
		yes = "[x] yes"
	} else {
		no = "[x] no"
	}
	return fmt.Sprintf("%s   %s", yes, no)
}

func checkDepsCmd(ctx context.Context) tea.Cmd {
	return func() tea.Msg {
		status := setup.CheckDependencies(ctx)
		return initMsg{status: status}
	}
}

func installDepsCmd(ctx context.Context, installer setup.Installer, deps []setup.Dependency) tea.Cmd {
	return func() tea.Msg {
		logger := &strings.Builder{}
		err := setup.Install(ctx, installer, deps, logger, logger)
		if err != nil {
			return initMsg{err: err}
		}
		status := setup.CheckDependencies(ctx)
		if len(status.Missing) > 0 {
			return initMsg{err: fmt.Errorf("dependencies still missing: %s", setup.MissingSummary(status))}
		}
		return initMsg{status: status}
	}
}

func saveConfigCmd(ctx context.Context, outputDir, configPath string) tea.Cmd {
	return func() tea.Msg {
		cfgPath, err := resolveConfigPath(configPath)
		if err != nil {
			return initMsg{err: err}
		}
		if strings.TrimSpace(outputDir) != "" {
			v := viper.New()
			v.Set("output_dir", strings.TrimSpace(outputDir))
			if err := v.WriteConfigAs(cfgPath); err != nil {
				return initMsg{err: err}
			}
		}
		return initMsg{}
	}
}

func resolveDefaultOutput(configPath string) string {
	cfgPath, err := resolveConfigPath(configPath)
	if err == nil {
		v := viper.New()
		v.SetConfigFile(cfgPath)
		if err := v.ReadInConfig(); err == nil {
			if val := strings.TrimSpace(v.GetString("output_dir")); val != "" {
				return val
			}
		}
	}
	if cwd, err := os.Getwd(); err == nil {
		return cwd
	}
	return "."
}

func resolveConfigPath(configPath string) (string, error) {
	if configPath != "" {
		return configPath, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".scdl.yaml"), nil
}
