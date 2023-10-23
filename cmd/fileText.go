package cmd

import (
	"encoding/base64"
	"fmt"
	"github.com/spf13/cobra"
	"strings"
)

var fileTextCmd = &cobra.Command{
	Use:   "file-text name text",
	Short: "Search for text inside a file across repositories",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 2 {
			return fmt.Errorf("2 arguments are required but %d were provided", len(args))
		}
		return nil
	},
	Run: fileTextExecute,
}

func init() {
	fileTextCmd.Flags().CountVarP(&verbose, "verbose", "v", "Make the operation more talkative")
	search = fileTextCmd.Flags().String("search", "", "Search criteria")
}

func fileTextExecute(_ *cobra.Command, args []string) {
	ch := make(chan ProjectFile)
	fileName := args[0]
	searchedText := args[1]

	go fileSearch(fileName, search, []string{"12.0", "13.0", "14.0", "15.0", "16.0"}, verbose, ch)
	for pj := range ch {
		content, err := base64.StdEncoding.DecodeString(pj.file.Content)
		if err != nil {
			continue
		}
		contentStr := string(content[:])

		if strings.Contains(contentStr, searchedText) {
			fmt.Printf("✅  Found %s in file %s in %s @%s\n", searchedText, pj.file.FileName, pj.project.NameWithNamespace, pj.file.Ref)
		}
	}
}
