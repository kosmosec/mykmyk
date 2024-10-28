package cmd

import (
	"fmt"
	"os"

	"github.com/kosmosec/mykmyk/internal/api"
	"github.com/kosmosec/mykmyk/internal/scanner"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

func NewScan() *cobra.Command {

	newScan := cobra.Command{
		Use:   "scan",
		Short: "Scan targets",
		Long:  "",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfgPath, err := cmd.Flags().GetString("config")
			if err != nil {
				return err
			}
			cfg, err := applyConfig(cfgPath)
			if err != nil {
				return err
			}
			err = scanner.Scan(cmd.Context(), cfg)
			if err != nil {
				return err
			}
			return nil
		},
	}

	return &newScan
}

func applyConfig(cfgPath string) (api.Config, error) {
	var cfg api.Config
	if cfgPath == "" {
		currentDir := os.Getenv("PWD")
		if currentDir == "" {
			return api.Config{}, errors.Errorf("No PWD in the ENV.")
		}
		cfgPath = fmt.Sprintf("%s/%s", currentDir, api.DefaultConfigName)
		if _, err := os.Stat(cfgPath); err != nil {
			homeDirectory := os.Getenv("HOME")
			if homeDirectory == "" {
				return api.Config{}, errors.Errorf("No HOME in the ENV.")
			}
			cfgPath = fmt.Sprintf("%s/.config/mykmyk/%s", homeDirectory, api.DefaultConfigName)
		}
	}

	rawCfg, err := os.ReadFile(cfgPath)
	if err != nil {
		return api.Config{}, err
	}
	err = yaml.Unmarshal(rawCfg, &cfg)
	if err != nil {
		return api.Config{}, err
	}
	return cfg, nil
}
