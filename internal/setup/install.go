package setup

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"runtime"
)

type Installer struct {
	Name string
	Cmd  []string
	Pre  []string
}

func DetectInstaller() (Installer, error) {
	if runtime.GOOS == "darwin" {
		if _, err := exec.LookPath("brew"); err == nil {
			return Installer{Name: "brew", Cmd: []string{"brew", "install"}}, nil
		}
		return Installer{}, fmt.Errorf("homebrew not found")
	}

	if runtime.GOOS == "linux" {
		if _, err := exec.LookPath("apt-get"); err == nil {
			return Installer{Name: "apt", Pre: []string{"sudo", "apt-get", "update"}, Cmd: []string{"sudo", "apt-get", "install", "-y"}}, nil
		}
		if _, err := exec.LookPath("dnf"); err == nil {
			return Installer{Name: "dnf", Cmd: []string{"sudo", "dnf", "install", "-y"}}, nil
		}
	}

	return Installer{}, fmt.Errorf("no supported package manager found")
}

func Install(ctx context.Context, installer Installer, deps []Dependency, stdout, stderr io.Writer) error {
	if len(deps) == 0 {
		return nil
	}

	pkgs := packagesFor(deps)
	if len(pkgs) == 0 {
		return nil
	}

	if len(installer.Pre) > 0 {
		if err := runCommand(ctx, installer.Pre, stdout, stderr); err != nil {
			return err
		}
	}

	cmd := append([]string{}, installer.Cmd...)
	cmd = append(cmd, pkgs...)
	return runCommand(ctx, cmd, stdout, stderr)
}

func packagesFor(deps []Dependency) []string {
	pkgs := make([]string, 0, len(deps))
	for _, dep := range deps {
		switch dep.Name {
		case "yt-dlp":
			pkgs = append(pkgs, "yt-dlp")
		case "ffmpeg":
			pkgs = append(pkgs, "ffmpeg")
		}
	}
	return pkgs
}

func runCommand(ctx context.Context, cmd []string, stdout, stderr io.Writer) error {
	if len(cmd) == 0 {
		return errors.New("empty command")
	}
	command := exec.CommandContext(ctx, cmd[0], cmd[1:]...)
	command.Stdout = stdout
	command.Stderr = stderr
	if err := command.Run(); err != nil {
		return fmt.Errorf("%s failed: %w", cmd[0], err)
	}
	return nil
}
