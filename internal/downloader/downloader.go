package downloader

import (
	"context"
	"fmt"
	"io"

	"scdl/internal/downloader/youtubedl"
	"scdl/internal/downloader/ytdlp"
)

type Tool interface {
	Name() string
	DownloadAudio(ctx context.Context, url, outputTemplate string, stdout, stderr io.Writer) error
	Available() bool
}

func Detect() (Tool, error) {
	if (ytdlp.Tool{}).Available() {
		return ytdlp.Tool{}, nil
	}
	if (youtubedl.Tool{}).Available() {
		return youtubedl.Tool{}, nil
	}
	return nil, fmt.Errorf("neither youtube-dl nor yt-dlp is installed. Please install one of them")
}
