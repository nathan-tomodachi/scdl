package setup

import (
	"context"
	"fmt"
	"os/exec"
)

type Dependency struct {
	Name        string
	Command     string
	VersionArgs []string
}

type Status struct {
	FFmpegOK   bool
	YtDlpOK    bool
	YtDlOK     bool
	Missing    []Dependency
	MissingYtD bool
}

func CheckDependencies(ctx context.Context) Status {
	ffmpeg := Dependency{Name: "ffmpeg", Command: "ffmpeg", VersionArgs: []string{"-version"}}
	ytdlp := Dependency{Name: "yt-dlp", Command: "yt-dlp", VersionArgs: []string{"--version"}}
	ytdl := Dependency{Name: "youtube-dl", Command: "youtube-dl", VersionArgs: []string{"--version"}}

	status := Status{}
	status.FFmpegOK = checkTool(ctx, ffmpeg)
	status.YtDlpOK = checkTool(ctx, ytdlp)
	status.YtDlOK = checkTool(ctx, ytdl)

	if !status.FFmpegOK {
		status.Missing = append(status.Missing, ffmpeg)
	}
	if !status.YtDlpOK && !status.YtDlOK {
		status.Missing = append(status.Missing, ytdlp)
		status.MissingYtD = true
	}

	return status
}

func checkTool(ctx context.Context, dep Dependency) bool {
	if _, err := exec.LookPath(dep.Command); err != nil {
		return false
	}
	if len(dep.VersionArgs) == 0 {
		return true
	}
	cmd := exec.CommandContext(ctx, dep.Command, dep.VersionArgs...)
	if err := cmd.Run(); err != nil {
		return false
	}
	return true
}

func MissingSummary(status Status) string {
	if len(status.Missing) == 0 {
		return ""
	}
	missing := make([]string, 0, len(status.Missing))
	for _, dep := range status.Missing {
		missing = append(missing, dep.Name)
	}
	return fmt.Sprintf("Missing: %s", join(missing, ", "))
}

func join(items []string, sep string) string {
	if len(items) == 0 {
		return ""
	}
	out := items[0]
	for i := 1; i < len(items); i++ {
		out += sep + items[i]
	}
	return out
}
