package internal

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/xanzy/go-gitlab"
)

func SafeNewClient() *gitlab.Client {
	git, err := gitlab.NewClient(viper.GetString("token"), gitlab.WithBaseURL(viper.GetString("url")))
	cobra.CheckErr(err)

	return git
}
