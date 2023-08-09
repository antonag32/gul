package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"regexp"
)

var validKey = regexp.MustCompile(`[a-z]`)

var configCmd = &cobra.Command{
	Use:   "config key [value]",
	Short: "Configure the application through key-value pairs",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return fmt.Errorf("key is a required argument")
		}
		if len(args) > 2 {
			return fmt.Errorf("2 arguments are required but %d were provided", len(args))
		}
		if !validKey.MatchString(args[0]) {
			return fmt.Errorf("key must only contains lower case characters (a-z)")
		}

		return nil
	},
	Run: configExecute,
}

func configExecute(_ *cobra.Command, args []string) {
	if len(args) == 2 {
		viper.Set(args[0], args[1])
	} else {
		fmt.Println(viper.GetString(args[0]))
	}
}
