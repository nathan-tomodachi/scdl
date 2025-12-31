package media

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"

	"scdl/internal/downloader"
	"scdl/internal/util"
)

func DownloadFile(ctx context.Context, url, dest string) error {
	if err := util.DownloadToFile(ctx, url, dest); err != nil {
		return fmt.Errorf("failed to download file: %w", err)
	}
	return nil
}

func DownloadAudio(ctx context.Context, tool downloader.Tool, url, outputPath string, stdout, stderr io.Writer) error {
	_ = os.Remove(outputPath)
	outputTemplate := outputPath + ".%(ext)s"

	if err := tool.DownloadAudio(ctx, url, outputTemplate, stdout, stderr); err != nil {
		return fmt.Errorf("failed to download audio: %w", err)
	}

	if info, err := os.Stat(outputPath); err == nil && info.Size() > 0 {
		return nil
	}

	matches, err := filepath.Glob(outputPath + ".*")
	if err != nil {
		return fmt.Errorf("failed to locate downloaded audio: %w", err)
	}
	if len(matches) == 0 {
		return fmt.Errorf("failed to download audio or downloaded file is empty")
	}

	sort.Strings(matches)
	found := matches[0]
	info, err := os.Stat(found)
	if err != nil || info.Size() == 0 {
		return fmt.Errorf("failed to download audio or downloaded file is empty")
	}

	if found != outputPath {
		if err := os.Rename(found, outputPath); err != nil {
			return fmt.Errorf("failed to rename downloaded audio: %w", err)
		}
	}
	return nil
}

func CreateVideo(ctx context.Context, imagePath, audioPath, outputPath string, stdout, stderr io.Writer) error {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return fmt.Errorf("ffmpeg not found in PATH")
	}

	args := []string{
		"-loop", "1",
		"-i", imagePath,
		"-i", audioPath,
		"-c:v", "libx264",
		"-pix_fmt", "yuv420p",
		"-colorspace", "bt709",
		"-tune", "stillimage",
		"-c:a", "copy",
		"-shortest",
		outputPath,
		"-y",
	}

	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg failed to create the video: %w", err)
	}
	return nil
}
