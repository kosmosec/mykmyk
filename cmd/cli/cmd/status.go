package cmd

import (
	"github.com/kosmosec/mykmyk/internal/status"
	"github.com/spf13/cobra"
)

func NewStatus() *cobra.Command {

	newStatus := cobra.Command{
		Use:   "status",
		Short: "Show the status of the tasks",
		Long:  "",
		RunE: func(cmd *cobra.Command, args []string) error {
			targetFile, err := cmd.Flags().GetString("target")
			if err != nil {
				return err
			}
			cfgPath, err := cmd.Flags().GetString("config")
			if err != nil {
				return err
			}
			cfg, err := applyConfig(cfgPath)
			if err != nil {
				return err
			}
			err = status.Status(cmd.Context(), targetFile, cfg)
			if err != nil {
				return err
			}
			return nil
		},
	}

	return &newStatus
}
