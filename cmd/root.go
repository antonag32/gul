package cmd

import (
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
	"path"
)

var rootCmd = &cobra.Command{
	Use:     "gul",
	Short:   "Command line based utilities for GitLab",
	Version: "0.0.1v",
}

func initConfig() {
	home, err := os.UserHomeDir()
	cobra.CheckErr(err)

	configPath := path.Join(home, ".config", "gul")
	cobra.CheckErr(os.MkdirAll(configPath, 0750))

	viper.AddConfigPath(configPath)
	viper.SetConfigName("gul")
	viper.SetConfigType("yaml")

	viper.SetDefault("token", "your-gitlab-token")
	viper.SetDefault("url", "https://gitlab.com")

	if err := viper.ReadInConfig(); err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if errors.As(err, &configFileNotFoundError) {
			cobra.CheckErr(viper.SafeWriteConfig())
		}
	}
}

func init() {
	initConfig()
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(fileCmd)
	rootCmd.AddCommand(fileTextCmd)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	cobra.CheckErr(viper.WriteConfig())
}
