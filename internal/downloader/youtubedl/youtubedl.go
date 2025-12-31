package youtubedl

import (
	"context"
	"io"
	"os/exec"
)

const CommandName = "youtube-dl"

var AudioArgs = []string{
	"--extract-audio",
	"--audio-format",
	"mp3",
}

type Tool struct{}

func (Tool) Name() string {
	return CommandName
}

func (Tool) Available() bool {
	_, err := exec.LookPath(CommandName)
	return err == nil
}

func (Tool) DownloadAudio(ctx context.Context, url, outputTemplate string, stdout, stderr io.Writer) error {
	args := append([]string{}, AudioArgs...)
	args = append(args, "-o", outputTemplate, url)

	cmd := exec.CommandContext(ctx, CommandName, args...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	return cmd.Run()
}
