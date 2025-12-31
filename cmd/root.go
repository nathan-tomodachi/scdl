package cmd

import (
	"context"
	"io"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"scdl/internal/app"
	"scdl/internal/prompt"
	"scdl/internal/tui"
)

var cfgFile string
var outputDir string
var forceOverwrite bool

var rootCmd = &cobra.Command{
	Use:   "scdl [soundcloud_url]",
	Short: "Download a SoundCloud track and render it as an MP4",
	Args:  cobra.RangeArgs(0, 1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return tui.Run(cmd.Context(), tui.Config{
				URL:       "",
				OutputDir: resolveOutputDir(outputDir),
				Force:     forceOverwrite,
			})
		}
		return runRoot(cmd.Context(), args[0], os.Stdin, os.Stdout)
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.scdl.yaml)")
	rootCmd.PersistentFlags().StringVarP(&outputDir, "output", "o", "", "output directory (default is current directory)")
	rootCmd.PersistentFlags().BoolVarP(&forceOverwrite, "force", "f", false, "overwrite output file if it exists")
}

func runRoot(ctx context.Context, url string, in io.Reader, out io.Writer) error {
	opts := app.Options{
		OutputDir: resolveOutputDir(outputDir),
		Force:     forceOverwrite,
		ConfirmOverwrite: func(path string) (bool, error) {
			return prompt.ConfirmOverwrite(path, in, out)
		},
	}
	return app.Run(ctx, url, opts, in, out)
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".scdl")
	}

	viper.AutomaticEnv()

	_ = viper.ReadInConfig()
}

func resolveOutputDir(flagValue string) string {
	if flagValue != "" {
		return flagValue
	}
	return viper.GetString("output_dir")
}
