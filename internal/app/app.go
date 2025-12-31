package app

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"scdl/internal/downloader"
	"scdl/internal/media"
	"scdl/internal/soundcloud"
	"scdl/internal/util"
)

type Options struct {
	OutputDir        string
	Force            bool
	ConfirmOverwrite func(path string) (bool, error)
	Status           func(message string)
}

func Run(ctx context.Context, rawURL string, opts Options, in io.Reader, out io.Writer) error {
	status := func(message string) {
		if opts.Status != nil {
			opts.Status(message)
		}
	}

	status("Fetching page data")
	url, err := soundcloud.NormalizeURL(rawURL)
	if err != nil {
		return err
	}

	html, err := soundcloud.FetchHTML(ctx, url)
	if err != nil {
		return err
	}

	status("Parsing metadata")
	titleResult, err := soundcloud.DeriveBaseFilename(html)
	if err != nil {
		return err
	}
	for _, warning := range titleResult.Warnings {
		fmt.Fprintln(out, warning)
	}
	switch titleResult.Source {
	case soundcloud.TitleSourceOG:
		fmt.Fprintf(out, "Using og:title for filename: %s\n", titleResult.Raw)
	case soundcloud.TitleSourceH1H2:
		fmt.Fprintf(out, "Using title from h1 ('%s') and artist from h2 ('%s').\n", titleResult.Song, titleResult.Artist)
	case soundcloud.TitleSourceTitleTag:
		fmt.Fprintf(out, "Warning: Using title from <title> tag: %s\n", titleResult.Raw)
	case soundcloud.TitleSourceOGRaw:
		fmt.Fprintf(out, "Warning: Using raw og:title as last resort: %s\n", titleResult.Raw)
	}

	baseFilename := util.SanitizeFilename(titleResult.Base)
	if baseFilename == "" {
		return fmt.Errorf("could not determine a base filename")
	}

	fmt.Fprintf(out, "Raw base filename: '%s'\n", baseFilename)
	baseFilename = util.CleanBaseFilename(baseFilename)
	fmt.Fprintf(out, "Cleaned base filename: '%s'\n", baseFilename)
	if baseFilename == "" {
		return fmt.Errorf("base filename became empty after cleanup")
	}

	outputDir := strings.TrimSpace(opts.OutputDir)
	if outputDir == "" {
		outputDir = "."
	}
	outputPath := filepath.Join(outputDir, baseFilename+".mp4")
	fmt.Fprintf(out, "Proposed output filename: %s\n", outputPath)
	status("Preparing output")

	if _, err := os.Stat(outputPath); err == nil {
		if opts.Force {
			fmt.Fprintln(out, "Overwriting existing file.")
		} else {
			if opts.ConfirmOverwrite == nil {
				return fmt.Errorf("output file already exists: %s (use --force to overwrite)", outputPath)
			}
			overwrite, err := opts.ConfirmOverwrite(outputPath)
			if err != nil {
				return err
			}
			if !overwrite {
				fmt.Fprintln(out, "Operation cancelled by user. The existing file was not changed.")
				return nil
			}
			fmt.Fprintln(out, "Overwriting existing file.")
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("unable to check output file: %w", err)
	}

	tmpDir, err := os.MkdirTemp("", "sc_temp.")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	imageTmp := filepath.Join(tmpDir, "sc_img.jpg")
	audioTmp := filepath.Join(tmpDir, "sc_audio.mp3")

	success := false
	defer func() {
		_ = os.RemoveAll(tmpDir)
		if !success && outputPath != "" {
			if _, err := os.Stat(outputPath); err == nil {
				fmt.Fprintf(out, "Removing incomplete output file: %s\n", outputPath)
				_ = os.Remove(outputPath)
			}
		}
	}()

	imageURL, err := soundcloud.ExtractImageURL(html)
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "Found image URL: %s\n", imageURL)

	tool, err := downloader.Detect()
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "Using %s to download audio.\n", tool.Name())

	status("Downloading image")
	fmt.Fprintln(out, "Downloading image...")
	if err := media.DownloadFile(ctx, imageURL, imageTmp); err != nil {
		return err
	}

	status("Downloading audio")
	fmt.Fprintln(out, "Downloading audio...")
	if err := media.DownloadAudio(ctx, tool, url, audioTmp, out, out); err != nil {
		return err
	}

	status("Creating video")
	fmt.Fprintf(out, "Creating video file '%s'...\n", outputPath)
	if err := media.CreateVideo(ctx, imageTmp, audioTmp, outputPath, out, out); err != nil {
		return err
	}

	fmt.Fprintf(out, "Output video created: %s\n", outputPath)
	status("Done")
	success = true
	return nil
}
