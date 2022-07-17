package cmd

import (
	"fmt"
	"github.com/jphastings/real-button/pkg"
	"github.com/spf13/cobra"
	"log"
	"os"
	"time"
)

var configFile string

var rootCmd = &cobra.Command{
	Use: "real-button",
	RunE: func(cmd *cobra.Command, args []string) error {
		app, err := pkg.Load(configFile)
		if err != nil {
			return err
		}

		for {
			if err := app.Run(); err != nil {
				fmt.Printf("Error running the button: %v", err)
				fmt.Printf("Will retry in 10 seconds")
				time.Sleep(10)
			}

		}

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
