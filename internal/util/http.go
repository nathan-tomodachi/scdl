package util

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"
)

const DefaultUserAgent = "scdl/1.0"

var httpClient = &http.Client{
	Timeout: 30 * time.Second,
}

func FetchText(ctx context.Context, url string) (string, error) {
	var lastErr error
	for attempt := 1; attempt <= 3; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return "", err
		}
		req.Header.Set("User-Agent", DefaultUserAgent)

		resp, err := httpClient.Do(req)
		if err != nil {
			lastErr = err
		} else {
			body, readErr := readResponse(resp)
			if readErr != nil {
				lastErr = readErr
			} else {
				return string(body), nil
			}
		}

		if !shouldRetry(ctx, lastErr) || attempt == 3 {
			break
		}
		sleepWithBackoff(ctx, attempt)
	}
	return "", fmt.Errorf("failed to fetch page data after retries: %w", lastErr)
}

func DownloadToFile(ctx context.Context, url, dest string) error {
	var lastErr error
	for attempt := 1; attempt <= 3; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return err
		}
		req.Header.Set("User-Agent", DefaultUserAgent)

		resp, err := httpClient.Do(req)
		if err != nil {
			lastErr = err
		} else {
			lastErr = saveResponse(resp, dest)
			if lastErr == nil {
				return nil
			}
		}

		if !shouldRetry(ctx, lastErr) || attempt == 3 {
			break
		}
		sleepWithBackoff(ctx, attempt)
	}
	return fmt.Errorf("failed to download file after retries: %w", lastErr)
}

func readResponse(resp *http.Response) ([]byte, error) {
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, statusError{code: resp.StatusCode, status: resp.Status}
	}
	return io.ReadAll(resp.Body)
}

func saveResponse(resp *http.Response, dest string) error {
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return statusError{code: resp.StatusCode, status: resp.Status}
	}

	file, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	return err
}

func shouldRetry(ctx context.Context, err error) bool {
	if err == nil {
		return false
	}
	if ctx.Err() != nil {
		return false
	}
	if errors.Is(err, io.EOF) {
		return true
	}
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		return true
	}
	var statusErr statusError
	if errors.As(err, &statusErr) {
		return statusErr.code == http.StatusTooManyRequests || statusErr.code >= 500
	}
	return false
}

func sleepWithBackoff(ctx context.Context, attempt int) {
	delay := time.Duration(attempt*attempt) * 500 * time.Millisecond
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
	case <-timer.C:
	}
}

type statusError struct {
	code   int
	status string
}

func (e statusError) Error() string {
	return fmt.Sprintf("unexpected status %s", e.status)
}
