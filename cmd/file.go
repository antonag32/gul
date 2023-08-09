package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/xanzy/go-gitlab"
	"gul/internal"
	"strings"
)

var verbose int

var fileCmd = &cobra.Command{
	Use:   "file name",
	Short: "Search for a file across repositories",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return fmt.Errorf("1 argument is required but %d were provided", len(args))
		}
		return nil
	},
	Run: fileExecute,
}

func init() {
	fileCmd.Flags().CountVarP(&verbose, "verbose", "v", "Make the operation more talkative")
}

func fileExecute(_ *cobra.Command, args []string) {
	client := internal.SafeNewClient()
	fileName := args[0]
	if verbose > 0 {
		fmt.Printf("URL Encoded filename is %s\n", gitlab.PathEscape(fileName))
	}

	pagOpts := gitlab.ListOptions{PerPage: 100, Page: 1}
	for {
		projects, resp, err := client.Projects.ListProjects(&gitlab.ListProjectsOptions{
			WithProgrammingLanguage: gitlab.String("Python"),
			ListOptions:             pagOpts,
		})
		cobra.CheckErr(err)

		if verbose > 0 {
			fmt.Printf("Page %d, searching in %d project(s)\n", pagOpts.Page, len(projects))
		}

		for _, proj := range projects {
			if strings.HasSuffix(proj.Namespace.Name, "-dev") {
				if verbose > 1 {
					fmt.Printf("Skipping %s\n", proj.NameWithNamespace)
				}
				continue
			}

			if verbose > 1 {
				fmt.Printf("Looking in %s\n", proj.NameWithNamespace)
			}
			if searchFile(client, proj, fileName) {
				fmt.Printf("✅  Found %s in %s\n", fileName, proj.NameWithNamespace)
			}
		}

		if resp.NextPage == 0 {
			break
		}
		pagOpts.Page = resp.NextPage
	}
}

func searchFile(client *gitlab.Client, project *gitlab.Project, fileName string) bool {
	file, _, _ := client.RepositoryFiles.GetFile(project.ID, fileName, &gitlab.GetFileOptions{Ref: gitlab.String(project.DefaultBranch)})

	return file != nil
}
