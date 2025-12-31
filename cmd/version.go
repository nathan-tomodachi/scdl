package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var version = "dev"

func SetVersion(v string) {
	if v != "" {
		version = v
	}
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the scdl version",
	RunE: func(cmd *cobra.Command, args []string) error {
		_, err := fmt.Fprintln(cmd.OutOrStdout(), version)
		return err
	},
}
