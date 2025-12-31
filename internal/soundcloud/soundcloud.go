package soundcloud

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"scdl/internal/util"
)

const (
	TitleSourceOG       = "og:title"
	TitleSourceH1H2     = "h1/h2"
	TitleSourceTitleTag = "title"
	TitleSourceOGRaw    = "og:title raw"
)

type TitleResult struct {
	Base     string
	Raw      string
	Source   string
	Song     string
	Artist   string
	Warnings []string
}

var (
	ogTitleRe1      = regexp.MustCompile(`(?i)property="og:title"\s+content="([^"]+)"`)
	ogTitleRe2      = regexp.MustCompile(`(?i)content="([^"]+)"\s+property="og:title"`)
	h1TitleRe       = regexp.MustCompile(`(?is)<h1[^>]*class="[^"]*soundTitle__title[^"]*"[^>]*>.*?<span[^>]*>([^<]*)</span>.*?</h1>`)
	h2ArtistRe      = regexp.MustCompile(`(?is)<h2[^>]*class="[^"]*soundTitle__username[^"]*"[^>]*>.*?<a[^>]*>([^<]*)</a>.*?</h2>`)
	titleTagRe      = regexp.MustCompile(`(?is)<title>([^<]*)</title>`)
	imageTagRe      = regexp.MustCompile(`(?i)<img[^>]+src="(https://[^"]*artworks[^"]*\.jpg)"`)
	ogImageRe1      = regexp.MustCompile(`(?i)property="og:image"\s+content="([^"]+)"`)
	ogImageRe2      = regexp.MustCompile(`(?i)content="([^"]+)"\s+property="og:image"`)
	twitterImageRe1 = regexp.MustCompile(`(?i)name="twitter:image"\s+content="([^"]+)"`)
	twitterImageRe2 = regexp.MustCompile(`(?i)content="([^"]+)"\s+name="twitter:image"`)
	artworkURLRe    = regexp.MustCompile(`(?i)"artwork_url"\s*:\s*"([^"]+)"`)
)

func NormalizeURL(raw string) (string, error) {
	url := strings.TrimSpace(raw)
	if url == "" {
		return "", fmt.Errorf("usage: provide a soundcloud URL")
	}
	if idx := strings.Index(url, "?in"); idx != -1 {
		url = url[:idx]
	}
	if strings.Contains(strings.ToLower(url), "/sets/") {
		return "", fmt.Errorf("this script does not support playlists. Please provide a single track URL")
	}
	return url, nil
}

func FetchHTML(ctx context.Context, url string) (string, error) {
	body, err := util.FetchText(ctx, url)
	if err != nil {
		return "", fmt.Errorf("failed to fetch page data: %w", err)
	}
	return body, nil
}

func DeriveBaseFilename(html string) (TitleResult, error) {
	var result TitleResult

	if isNotFoundPage(html) {
		return TitleResult{}, fmt.Errorf("soundcloud track not found; check the URL")
	}

	ogTitle := matchFirst(ogTitleRe1, ogTitleRe2, html)
	if ogTitle == "" || !strings.Contains(ogTitle, " by ") {
		result.Warnings = append(result.Warnings, "Warning: Could not reliably parse 'Song Name by Artist Name' from og:title. Trying direct HTML tags...")
	}
	if ogTitle != "" && strings.Contains(ogTitle, " by ") {
		result.Raw = ogTitle
		result.Source = TitleSourceOG
		result.Base = strings.Replace(ogTitle, " by ", " - ", 1)
		return result, nil
	}

	song := strings.TrimSpace(matchFirst(h1TitleRe, nil, html))
	artist := strings.TrimSpace(matchFirst(h2ArtistRe, nil, html))
	if song != "" && artist != "" {
		result.Source = TitleSourceH1H2
		result.Song = song
		result.Artist = artist
		result.Base = fmt.Sprintf("%s - %s", song, artist)
		return result, nil
	}
	result.Warnings = append(result.Warnings, "Warning: Could not extract title/artist from h1/h2 tags.")

	titleTag := strings.TrimSpace(matchFirst(titleTagRe, nil, html))
	if titleTag != "" {
		if isGenericSoundCloudTitle(titleTag) {
			return TitleResult{}, fmt.Errorf("soundcloud URL does not point to a track")
		}
		cleaned := strings.ReplaceAll(titleTag, " | SoundCloud", "")
		cleaned = strings.ReplaceAll(cleaned, " Listen online", "")
		cleaned = strings.Replace(cleaned, " by ", " - ", 1)
		cleaned = strings.TrimSpace(cleaned)
		if cleaned != "" {
			result.Raw = cleaned
			result.Source = TitleSourceTitleTag
			result.Base = cleaned
			return result, nil
		}
	}

	if ogTitle != "" {
		result.Raw = ogTitle
		result.Source = TitleSourceOGRaw
		result.Base = ogTitle
		return result, nil
	}

	return TitleResult{}, fmt.Errorf("could not determine a base filename")
}

func ExtractImageURL(html string) (string, error) {
	imageURL := matchFirst(imageTagRe, nil, html)
	if imageURL == "" {
		imageURL = matchFirst(ogImageRe1, ogImageRe2, html)
	}
	if imageURL == "" {
		imageURL = matchFirst(twitterImageRe1, twitterImageRe2, html)
	}
	if imageURL == "" {
		imageURL = matchFirst(artworkURLRe, nil, html)
		imageURL = unescapeURL(imageURL)
	}
	if imageURL == "" {
		return "", fmt.Errorf("could not find image URL in the page")
	}
	return imageURL, nil
}

func matchFirst(primary *regexp.Regexp, secondary *regexp.Regexp, input string) string {
	if primary != nil {
		matches := primary.FindStringSubmatch(input)
		if len(matches) > 1 && matches[1] != "" {
			return matches[1]
		}
	}
	if secondary != nil {
		matches := secondary.FindStringSubmatch(input)
		if len(matches) > 1 && matches[1] != "" {
			return matches[1]
		}
	}
	return ""
}

func unescapeURL(value string) string {
	if value == "" || value == "null" {
		return ""
	}
	value = strings.ReplaceAll(value, `\u0026`, "&")
	value = strings.ReplaceAll(value, `\/`, "/")
	return value
}

func isNotFoundPage(html string) bool {
	lower := strings.ToLower(html)
	return strings.Contains(lower, "we can't find that sound") ||
		strings.Contains(lower, "sorry, we can't find that sound") ||
		strings.Contains(lower, "this track was not found")
}

func isGenericSoundCloudTitle(title string) bool {
	lower := strings.ToLower(strings.TrimSpace(title))
	return strings.Contains(lower, "soundcloud - hear the world")
}
