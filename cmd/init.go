package cmd

import (
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"scdl/internal/tui"
)

func init() {
	rootCmd.AddCommand(initCmd)
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Run the setup walkthrough",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfgPath := ""
		if cfgFile != "" {
			cfgPath = cfgFile
		} else {
			home, err := os.UserHomeDir()
			if err == nil {
				cfgPath = filepath.Join(home, ".scdl.yaml")
			}
		}
		return tui.RunInit(cmd.Context(), tui.InitConfig{ConfigPath: cfgPath})
	},
}
