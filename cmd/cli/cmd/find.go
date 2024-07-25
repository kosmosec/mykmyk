package cmd

import (
	"github.com/kosmosec/mykmyk/internal/finder"
	"github.com/spf13/cobra"
)

func NewFind() *cobra.Command {

	newFind := cobra.Command{
		Use:   "find",
		Short: "Find using filter",
		Long:  "",
		RunE: func(cmd *cobra.Command, args []string) error {
			targetFile, err := cmd.Flags().GetString("target")
			if err != nil {
				return err
			}
			portFilter, err := cmd.Flags().GetString("port")
			if err != nil {
				return err
			}
			err = finder.Find(cmd.Context(), targetFile, portFilter)
			if err != nil {
				return err
			}
			return nil
		},
	}

	return &newFind
}
