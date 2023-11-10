package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type configuration struct {
	noWork    bool
	noCommit  bool
	noPush    bool
	userName  string
	userEmail string
	sshDomain string
}

var pushCmd = &cobra.Command{
	Use:   "push workdir",
	Short: "Execute a mass push operation with instructions found on workdir",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return fmt.Errorf("workdir is required")
		}
		stat, err := os.Stat(args[0])
		if err != nil || !stat.IsDir() {
			return fmt.Errorf("invalid workdir: %s", args[0])
		}
		return nil
	},
	Run:                mergeExecute,
	DisableFlagParsing: true,
}

func mergeExecute(cmd *cobra.Command, args []string) {
	listFlag := cmd.Flags().Bool("list", false, "List all the projects that will be worked on")
	targetCount := cmd.Flags().Int("count", -1, "Amount of targets that will be processed, -1 for all")

	noWork := cmd.Flags().Bool("no-work", false, "Clone the repositories but do no other job")
	noCommit := cmd.Flags().Bool("no-commit", false, "Stop before committing")
	noPush := cmd.Flags().Bool("no-push", false, "Stop before pushing changes upstream")

	cobra.CheckErr(cmd.Flags().Parse(args))
	var cmdConfig = configuration{
		noWork:    *noWork,
		noCommit:  *noCommit,
		noPush:    *noPush,
		userName:  viper.GetString("user.name"),
		userEmail: viper.GetString("user.email"),
		sshDomain: viper.GetString("ssh.domain"),
	}

	workdir := args[0]
	targets := readTargets(path.Join(workdir, "targets.txt"))
	if *listFlag {
		fmt.Println("")
		for _, target := range targets {
			fmt.Println(target)
		}
		return
	}

	jobFile := path.Join(workdir, "job")
	_, err := os.Stat(jobFile)
	cobra.CheckErr(err)

	commitMsg := readCommitMessage(path.Join(workdir, "message.txt"))

	var waitGroup sync.WaitGroup
	var targetCh = make(chan string)
	for i := 0; i < 8; i++ {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			processTarget(workdir, jobFile, commitMsg, cmdConfig, targetCh)
		}()
	}

	if *targetCount > 0 {
		for i := 0; i < *targetCount; i++ {
			targetCh <- targets[i]
		}
	} else {
		for _, target := range targets {
			targetCh <- target
		}
	}

	close(targetCh)
	waitGroup.Wait()
}

func processTarget(workdir string, jobFile string, commitMsg string, gulConfig configuration, targets <-chan string) {
	for target := range targets {
		fmt.Printf("🔨  %s: setting up\n", target)
		branch := strings.Split(target, "@")[1]
		project := strings.Split(target, "@")[0]
		projectPath := path.Join(workdir, targetToDirName(target))
		newBranch := fmt.Sprintf("%s-%s", branch, filepath.Base(filepath.Dir(projectPath)))

		repo, worktree, err := setupProject(gulConfig.sshDomain, projectPath, project, branch, newBranch)
		if err != nil {
			printError(target, err)
			continue
		}
		if gulConfig.noWork {
			continue
		}

		fmt.Printf("⚙  %s: executing job\n", target)
		err = runJob(jobFile, projectPath)
		fmt.Printf("🏁  %s: ", target)
		if err != nil {
			fmt.Printf("%s\n", err)
		} else {
			fmt.Printf("exit status 0\n")
		}

		if gulConfig.noCommit || err == nil {
			continue
		}

		hash, err := worktree.Commit(commitMsg, &git.CommitOptions{
			All:               true,
			AllowEmptyCommits: false,
			Author: &object.Signature{
				Name:  gulConfig.userName,
				Email: gulConfig.userEmail,
				When:  time.Now(),
			},
		})
		if err != nil {
			printError(target, err)
			continue
		}
		fmt.Printf("💾  %s: commited %s\n", target, hash)

		if gulConfig.noPush {
			continue
		}

		ref := config.RefSpec(
			plumbing.NewBranchReferenceName(newBranch).String() +
				":" +
				plumbing.NewBranchReferenceName(newBranch).String(),
		)
		err = repo.Push(&git.PushOptions{
			RemoteName: "dev",
			RefSpecs:   []config.RefSpec{ref},
		})
		if err != nil && strings.Contains(err.Error(), "not be found") {
			err = repo.Push(&git.PushOptions{
				RemoteName: "origin",
				RefSpecs:   []config.RefSpec{ref},
			})
			if err != nil {
				printError(target, err)
				continue
			}
			fmt.Printf("☁  %s: pushed to origin\n", target)
		} else {
			fmt.Printf("☁  %s: pushed to dev\n", target)
		}

	}
}

func runJob(jobFile string, projectPath string) error {
	jobCmd := exec.Command(jobFile)
	jobCmd.Dir = projectPath

	return jobCmd.Run()
}

// Clone a project and checkout to a new branch, preparing the project for future work. Also add a "-dev" remote.
// In case the project is already cloned or the branch already exists no extra work is done.
// In case of error the function returns immediately, the cases above are not considered errors.
func setupProject(domain string, path string, project string, branch string, newBranch string) (*git.Repository, *git.Worktree, error) {
	var repo *git.Repository
	var err error

	repo, err = git.PlainClone(
		path,
		false,
		&git.CloneOptions{
			URL:           fmt.Sprintf("git@%s:%s.git", domain, project),
			ReferenceName: plumbing.NewBranchReferenceName(branch),
			Depth:         1,
		},
	)

	if errors.Is(err, git.ErrRepositoryAlreadyExists) {
		repo, err = git.PlainOpen(path)
	} else if err != nil {
		return nil, nil, err
	}

	names := strings.Split(project, "/")
	devName := names[0] + "-dev/" + names[1]
	sshDevURL := fmt.Sprintf("git@%s:%s.git", domain, devName)
	_, remoteErr := repo.CreateRemote(
		&config.RemoteConfig{
			Name: "dev",
			URLs: []string{sshDevURL},
		})

	if err != nil && !errors.Is(remoteErr, git.ErrRemoteExists) {
		return nil, nil, err
	}

	worktree, err := repo.Worktree()
	if err != nil {
		return nil, nil, err
	}

	err = worktree.Checkout(&git.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName(newBranch),
		Create: true,
	})
	if err != nil && strings.Contains(err.Error(), "already exists") {
		return repo, worktree, nil
	}

	return repo, worktree, err
}

func readCommitMessage(filename string) string {
	file, err := os.Open(filename)
	cobra.CheckErr(err)
	defer file.Close()

	var body string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		body += scanner.Text() + "\n"
	}
	cobra.CheckErr(scanner.Err())

	return body
}

func readTargets(filename string) []string {
	file, err := os.Open(filename)
	cobra.CheckErr(err)
	defer file.Close()

	var targets []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		target := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(target, "#") {
			continue
		}

		targets = append(targets, target)
	}
	cobra.CheckErr(scanner.Err())

	return targets
}

func targetToDirName(target string) string {
	retVal := strings.ReplaceAll(target, "@", "-")
	retVal = strings.ReplaceAll(retVal, "/", "-")

	return retVal
}

func printError(target string, err error) {
	fmt.Printf("❌  %s: %s\n", target, err)
}
