package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/xanzy/go-gitlab"
	"gul/internal"
	"strings"
)

type ProjectFile struct {
	file    *gitlab.File
	project *gitlab.Project
}

var verbose int
var search *string

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
	search = fileCmd.Flags().String("search", "", "Search criteria")
}

func fileExecute(_ *cobra.Command, args []string) {
	ch := make(chan ProjectFile)
	go fileSearch(args[0], search, nil, verbose, ch)
	for pj := range ch {
		fmt.Printf("✅  Found %s in %s\n", pj.file.FileName, pj.project.NameWithNamespace)
	}
}

func fileSearch(fileName string, search *string, branches []string, verbose int, channel chan ProjectFile) {
	client := internal.SafeNewClient()
	if verbose > 0 {
		fmt.Printf("URL Encoded filename is %s\n", gitlab.PathEscape(fileName))
	}

	pagOpts := gitlab.ListOptions{PerPage: 100, Page: 1}
	for {
		projects, resp, err := client.Projects.ListProjects(&gitlab.ListProjectsOptions{
			WithProgrammingLanguage: gitlab.String("Python"),
			ListOptions:             pagOpts,
			Search:                  search,
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

			if branches == nil {
				branches = []string{proj.DefaultBranch}
			}

			for _, branch := range branches {
				if verbose > 1 {
					fmt.Printf("Looking in %s@%s\n", proj.NameWithNamespace, branch)
				}
				file, _, _ := client.RepositoryFiles.GetFile(
					proj.ID, fileName,
					&gitlab.GetFileOptions{Ref: gitlab.String(branch)},
				)
				if file != nil {
					channel <- ProjectFile{file, proj}
				}
			}

		}

		if resp.NextPage == 0 {
			break
		}
		pagOpts.Page = resp.NextPage
	}

	close(channel)
}
