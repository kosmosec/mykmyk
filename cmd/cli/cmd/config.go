package cmd

import (
	_ "embed"
	"errors"
	"fmt"
	"os"

	"github.com/kosmosec/mykmyk/internal/api"
	"github.com/spf13/cobra"
)

//go:embed config.yml
var defaultCfg []byte

func NewConfig() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Create config",
		Long:  "Create default config for mykmyk",
		RunE: func(cmd *cobra.Command, args []string) error {
			err := creatLocalConfig()
			if err != nil {
				return err
			}
			err = createGlobalConfig()
			if err != nil {
				return err
			}
			return nil
		},
	}
}

func creatLocalConfig() error {
	err := os.WriteFile(api.DefaultConfigName, defaultCfg, 0664)
	if err != nil {
		return err
	}

	return nil
}

func createGlobalConfig() error {
	homeDirectory := os.Getenv("HOME")
	configPath := fmt.Sprintf("%s/.config/mykmyk", homeDirectory)
	if _, err := os.Stat(configPath); errors.Is(err, os.ErrNotExist) {
		err := os.MkdirAll(configPath, os.ModePerm)
		if err != nil {
			return err
		}
	}

	pathToConfigFile := fmt.Sprintf("%s/%s", configPath, api.DefaultConfigName)
	if _, err := os.Stat(pathToConfigFile); errors.Is(err, os.ErrNotExist) {
		err := os.WriteFile(pathToConfigFile, defaultCfg, 0664)
		if err != nil {
			return err
		}
	}
	return nil
}
