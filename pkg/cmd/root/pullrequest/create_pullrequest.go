package pullrequest

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cmdrunner"
	"github.com/jenkins-x/jx-helpers/v3/pkg/gitclient"
	"github.com/jenkins-x/jx-helpers/v3/pkg/gitclient/cli"
	"github.com/jenkins-x/jx-helpers/v3/pkg/gitclient/gitdiscovery"
	"github.com/jenkins-x/jx-helpers/v3/pkg/input"
	"github.com/jenkins-x/jx-helpers/v3/pkg/input/survey"
	"github.com/jenkins-x/jx-helpers/v3/pkg/options"
	"github.com/jenkins-x/jx-helpers/v3/pkg/scmhelpers"
	"github.com/jenkins-x/jx-promote/pkg/envctx"
	"github.com/jenkins-x/jx-promote/pkg/environments"

	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/helper"

	"github.com/pkg/errors"

	"github.com/spf13/cobra"

	"fmt"

	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/templates"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
)

const (
	optionTitle = "title"
)

var (
	createPullRequestLong = templates.LongDesc(`
		Creates a Pull Request in a the git project of the current directory. 

		If --push is specified the contents of the directory will be committed, pushed and used to create the pull request  
`)

	createPullRequestExample = templates.Examples(`
		# Create a Pull Request in the current project
		jx project pullrequest -t "my PR title"


		# Create a Pull Request with a title and a body
		jx project pullrequest -t "my PR title" --body "	
		some more
		text
		goes
		here
		""
"
	`)
)

// CreatePullRequestOptions the options for thecommand
type CreatePullRequestOptions struct {
	scmhelpers.Options

	BatchMode bool
	Title     string
	Body      string
	Labels    []string
	Base      string
	Push      bool
	Fork      bool

	Input   input.Interface
	Results *scm.PullRequest
}

// NewCmdCreatePullRequest creates a command object for the "create" command
func NewCmdCreatePullRequest() *cobra.Command {
	options := &CreatePullRequestOptions{}

	cmd := &cobra.Command{
		Use:     "pullrequest",
		Short:   "Create a Pull Request on the git project for the current directory",
		Aliases: []string{"pr", "pull request"},
		Long:    createPullRequestLong,
		Example: createPullRequestExample,
		Run: func(cmd *cobra.Command, args []string) {
			err := options.Run()
			helper.CheckErr(err)
		},
	}

	//cmd.Flags().StringVarP(&options.Dir, "dir", "", "", "The source directory used to detect the Git repository. Defaults to the current directory")
	cmd.Flags().StringVarP(&options.Title, optionTitle, "t", "", "The title of the pullrequest to create")
	cmd.Flags().StringVarP(&options.Body, "body", "", "", "The body of the pullrequest")
	cmd.Flags().StringVarP(&options.Base, "base", "", "master", "The base branch to create the pull request into")
	cmd.Flags().StringArrayVarP(&options.Labels, "label", "l", []string{}, "The labels to add to the pullrequest")
	cmd.Flags().BoolVarP(&options.Push, "push", "", false, "If true the contents of the source directory will be committed, pushed, and used to create the pull request")
	cmd.Flags().BoolVarP(&options.Fork, "fork", "", false, "If true, and the username configured to push the repo is different from the org name a PR is being created against, assume that this is a fork")

	cmd.Flags().BoolVarP(&options.BatchMode, "batch-mode", "b", false, "Enables batch mode which avoids prompting for user input")

	options.Options.AddFlags(cmd)
	return cmd
}

func (o *CreatePullRequestOptions) Validate() error {
	if o.CommandRunner == nil {
		o.CommandRunner = cmdrunner.QuietCommandRunner
	}
	err := o.Options.Validate()
	if err != nil {
		return errors.Wrapf(err, "failed to validate options")
	}
	if o.Input == nil {
		o.Input = survey.NewInput()
	}
	if o.GitClient == nil {
		o.GitClient = cli.NewCLIClient("", o.CommandRunner)
	}
	return nil
}

// Run implements the command
func (o *CreatePullRequestOptions) Run() error {
	err := o.Validate()
	if err != nil {
		return errors.Wrapf(err, "failed to validate options")
	}

	// lets discover the git dir
	if o.Dir == "" {
		dir, err := os.Getwd()
		if err != nil {
			return errors.Wrapf(err, "getting working directory")
		}
		o.Dir = dir
	}
	scmClient := o.ScmClient
	gitInfo := o.GitURL

	ctx := context.Background()
	fullName := o.FullRepositoryName
	_, _, err = scmClient.Repositories.Find(ctx, fullName)
	if err != nil {
		return errors.Wrapf(err, "failed to find repository %s", fullName)
	}

	user, _, err := scmClient.Users.Find(ctx)
	if err != nil {
		return errors.Wrapf(err, "failed to find the current user")
	}

	// Rebuild the gitInfo so that we get all the info we need
	currentUser := user.Login
	if o.Fork && currentUser != gitInfo.Organisation {
		forkName := scm.Join(currentUser, gitInfo.Name)
		_, _, err = scmClient.Repositories.Find(ctx, forkName)
		if err != nil {
			return errors.Wrapf(err, "failed to find repository %s, does the fork exist? Try running without --fork", forkName)
		}
	}

	po := &environments.EnvironmentPullRequestOptions{
		DevEnvContext: envctx.EnvironmentContext{},
		ScmClientFactory: scmhelpers.Factory{
			GitKind:      o.GitKind,
			GitServerURL: o.GitServerURL,
			GitToken:     o.GitToken,
			ScmClient:    o.ScmClient,
		},
		Gitter:        o.GitClient,
		CommandRunner: o.CommandRunner,
		GitKind:       o.GitKind,
		OutDir:        "",
		Function:      nil,
		Labels:        o.Labels,
		BranchName:    "",
		ScmClient:     o.ScmClient,
		BatchMode:     o.BatchMode,
		Fork:          o.Fork,
		CommitTitle:   o.Title,
		CommitMessage: o.Body,
	}

	err = o.createPullRequestDetails(po)
	if err != nil {
		return errors.Wrapf(err, "failed to create the PR details")
	}

	dir, err := filepath.Abs(o.Dir)
	if err != nil {
		return errors.Wrapf(err, "failed to find absolute dir of %s", o.Dir)
	}

	log.Logger().Debugf("creating pull request on %s in dir %s", o.SourceURL, dir)
	o.Results, err = po.CreatePullRequest(scmClient, o.SourceURL, fullName, o.Dir, true)
	if err != nil {
		return errors.Wrapf(err, "failed to create PR")
	}
	return nil
}

func (o *CreatePullRequestOptions) createPullRequestDetails(po *environments.EnvironmentPullRequestOptions) error {
	title := o.Title
	if title == "" {
		if o.BatchMode {
			return options.MissingOption(optionTitle)
		}
		defaultValue, body, err := o.findLastCommitTitle()
		if err != nil {
			log.Logger().Warnf("Failed to find last git commit title: %s", err)
		}
		if po.CommitMessage == "" {
			po.CommitMessage = body
		}
		po.CommitTitle, err = o.Input.PickValue("PullRequest title:", defaultValue, true, "")
		if err != nil {
			return err
		}
	}
	if title == "" {
		return fmt.Errorf("no title specified")
	}
	if po.BranchName == "" {
		var err error
		po.BranchName, err = gitclient.Branch(o.GitClient, o.Dir)
		if err != nil {
			return err
		}
	}
	return nil
}

func (o *CreatePullRequestOptions) findLastCommitTitle() (string, string, error) {
	title := ""
	body := ""
	dir := o.Dir
	_, err := gitdiscovery.FindGitURLFromDir(dir)
	if err != nil {
		return title, body, errors.Wrapf(err, "Failed to find git config in dir %s", dir)
	}
	message, err := gitclient.GetLatestCommitMessage(o.GitClient, dir)
	if err != nil {
		return title, body, err
	}
	lines := strings.SplitN(message, "\n", 2)
	if len(lines) < 2 {
		return message, "", nil
	}
	return lines[0], lines[1], nil
}
