package cmd

import (
	"encoding/base64"
	"fmt"
	"github.com/spf13/cobra"
	"strings"
)

var fileTextCmd = &cobra.Command{
	Use:                "file-text name text",
	Short:              "Search for text inside a file across repositories",
	Args:               cobra.ExactArgs(2),
	Run:                fileTextExecute,
	DisableFlagParsing: true,
}

func fileTextExecute(cmd *cobra.Command, args []string) {
	var verbose = cmd.Flags().CountP("verbose", "v", "Make the operation more talkative")
	var search = cmd.Flags().String("search", "", "Search criteria")
	cobra.CheckErr(cmd.Flags().Parse(args))

	ch := make(chan ProjectFile)
	fileName := args[0]
	searchedText := args[1]

	go fileSearch(fileName, search, []string{"12.0", "13.0", "14.0", "15.0", "16.0"}, *verbose, ch)
	for pj := range ch {
		content, err := base64.StdEncoding.DecodeString(pj.file.Content)
		if err != nil {
			continue
		}
		contentStr := string(content)

		if strings.Contains(contentStr, searchedText) {
			fmt.Printf("✅  Found %s in file %s in %s @%s\n", searchedText, pj.file.FileName, pj.project.NameWithNamespace, pj.file.Ref)
		}
	}
}
