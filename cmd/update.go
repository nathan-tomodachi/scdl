package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

const installScriptURL = "https://raw.githubusercontent.com/nathan-tomodachi/scdl/master/install.sh"
const tagsAPIURL = "https://api.github.com/repos/nathan-tomodachi/scdl/tags?per_page=1"

var updateVersion string

func init() {
	updateCmd.Flags().StringVar(&updateVersion, "version", "", "install a specific version (e.g. 1.0.0)")
	rootCmd.AddCommand(updateCmd)
}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update scdl to the latest release",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runUpdate(cmd, updateVersion)
	},
}

func runUpdate(cmd *cobra.Command, desiredVersion string) error {
	if desiredVersion == "" {
		latest, err := fetchLatestTag(cmd.Context())
		if err == nil && latest != "" && isUpToDate(version, latest) {
			_, werr := fmt.Fprintf(cmd.OutOrStdout(), "Already up to date (%s)\n", latest)
			return werr
		}
	}

	cmdline := fmt.Sprintf("curl -fsSL %s | sh", installScriptURL)
	command := exec.CommandContext(cmd.Context(), "sh", "-c", cmdline)
	command.Stdout = cmd.OutOrStdout()
	command.Stderr = cmd.ErrOrStderr()
	command.Env = os.Environ()
	if desiredVersion != "" {
		command.Env = append(command.Env, "SCDL_VERSION="+desiredVersion)
	}
	if err := command.Run(); err != nil {
		return fmt.Errorf("update failed: %w", err)
	}
	return nil
}

func fetchLatestTag(ctx context.Context) (string, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, tagsAPIURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "scdl")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status: %s", resp.Status)
	}

	var tags []struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tags); err != nil {
		return "", err
	}
	if len(tags) == 0 || strings.TrimSpace(tags[0].Name) == "" {
		return "", errors.New("no tags found")
	}
	return tags[0].Name, nil
}

func isUpToDate(current, latest string) bool {
	cur := strings.TrimPrefix(strings.TrimSpace(current), "v")
	lat := strings.TrimPrefix(strings.TrimSpace(latest), "v")
	if cur == "" || lat == "" {
		return false
	}
	return cur == lat
}
