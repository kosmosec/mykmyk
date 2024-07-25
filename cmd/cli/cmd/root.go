package cmd

import (
	"log"

	"github.com/spf13/cobra"
)

var (
	configPath string
	targetFile string
	portFilter string
)

func NewRoot() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "mykmyk",
		Short: "Nice blackbox pentesting CLI",
		Long:  "A utility that do the blackbox pentesting",
		Run: func(cmd *cobra.Command, args []string) {
			if err := cmd.Help(); err != nil {
				log.Fatalln("root cobra", err)
			}
		},
	}

	rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "", "path to the YAML config file")
	rootCmd.PersistentFlags().StringVarP(&targetFile, "target", "t", "", "path to the file with targets")
	rootCmd.PersistentFlags().StringVarP(&portFilter, "port", "p", "", "filter by port")

	rootCmd.AddCommand(
		NewScan(),
		NewConfig(),
		NewFind(),
		NewStatus(),
	)

	return rootCmd
}
