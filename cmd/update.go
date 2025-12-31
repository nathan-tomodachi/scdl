package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

const installScriptURL = "https://raw.githubusercontent.com/nathan-tomodachi/scdl/master/install.sh"

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

func runUpdate(cmd *cobra.Command, version string) error {
	cmdline := fmt.Sprintf("curl -fsSL %s | sh", installScriptURL)
	command := exec.CommandContext(cmd.Context(), "sh", "-c", cmdline)
	command.Stdout = cmd.OutOrStdout()
	command.Stderr = cmd.ErrOrStderr()
	command.Env = os.Environ()
	if version != "" {
		command.Env = append(command.Env, "SCDL_VERSION="+version)
	}
	if err := command.Run(); err != nil {
		return fmt.Errorf("update failed: %w", err)
	}
	return nil
}
