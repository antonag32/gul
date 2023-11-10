# gul
Command line based utilities for Gitlab

## Available tools

### file: search for repos containing a given file

### file-text: search for repos with given text in a file

### push: batch pushes to repositories after executing a given workflow
This tool is based on the concept of workflows. Each workflow consists of directory containing the following three files:

* targets.txt: list of repositories in format `namespace/project@branch` to perform the work on
* job: this executable will be run on each cloned repository, changes performed by it will be committed and pushed
* message.txt: commit message to be used

To use the tool you first need to configure the following with `gul config`:

1. Set a git user
2. Set a git email
3. Set a git ssh.domain

## Notes
Code here is really radioactive, I tried to integrate some linting into it to keep from going
too spaghetti, but there are no unit tests, just real life testing, 