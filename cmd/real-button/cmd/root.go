package cmd

import (
	"github.com/jphastings/real-button/pkg"
	"github.com/spf13/cobra"
	"log"
	"os"
)

var configFile string

var rootCmd = &cobra.Command{
	Use: "real-button",
	RunE: func(cmd *cobra.Command, args []string) error {
		app, err := pkg.Load(configFile)
		if err != nil {
			return err
		}

		return app.Run()
	},
}

func Execute(version string) {
	rootCmd.Version = version
	if err := rootCmd.Execute(); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().StringVarP(&configFile, "config", "", "config.yaml", "the config file to use")
}
