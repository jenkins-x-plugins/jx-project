package importcmd

import (
	"path/filepath"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/jx-gitops/pkg/cmd/repository/add"
	"github.com/jenkins-x/jx-helpers/pkg/kube/jxenv"
	"github.com/jenkins-x/jx-promote/pkg/environments"
	"github.com/pkg/errors"
)

func (o *ImportOptions) addSourceConfigPullRequest(gitURL string, gitKind string) error {
	if o.NoDevPullRequest {
		return nil
	}
	devEnv, err := jxenv.GetDevEnvironment(o.JXClient, o.Namespace)
	if err != nil {
		return errors.Wrapf(err, "failed to find the dev Environment")
	}

	devGitURL := devEnv.Spec.Source.URL
	if devGitURL == "" {
		return errors.Errorf("no git source URL for Environment %s", devEnv.Name)
	}

	// lets generate a PR
	base := devEnv.Spec.Source.Ref
	if base == "" {
		base = "master"
	}

	if o.SchedulerName == "" {
		g := filepath.Join(o.Dir, ".lighthouse", "*", "triggers.yaml")
		matches, err := filepath.Glob(g)
		if err != nil {
			return errors.Wrapf(err, "failed to evaluate glob %s", g)
		}
		if len(matches) > 0 {
			o.SchedulerName = "in-repo"
		}
	}

	pro := &environments.EnvironmentPullRequestOptions{
		ScmClientFactory:  o.ScmFactory,
		Gitter:            o.Git(),
		CommandRunner:     o.CommandRunner,
		GitKind:           o.ScmFactory.GitKind,
		OutDir:            "",
		BranchName:        "",
		PullRequestNumber: 0,
		CommitTitle:       "fix: import repository",
		CommitMessage:     "",
		ScmClient:         o.ScmFactory.ScmClient,
		BatchMode:         o.BatchMode,
		UseGitHubOAuth:    false,
		Fork:              false,
	}

	pro.Function = func() error {
		dir := pro.OutDir
		_, ao := add.NewCmdAddRepository()
		ao.Args = []string{gitURL}
		ao.Dir = dir
		ao.JXClient = o.JXClient
		ao.Namespace = o.Namespace
		ao.Scheduler = o.SchedulerName
		err := ao.Run()
		if err != nil {
			return errors.Wrapf(err, "failed to add git URL %s to the source-config.yaml file", gitURL)
		}

		err = o.modifyDevEnvironmentSource(o.Dir, dir, o.gitInfo, gitURL, gitKind)
		if err != nil {
			return errors.Wrapf(err, "failed to modify remote cluster")
		}
		return nil
	}

	/** TODO
	if pro.Username == "" {
		pro.Username = o.getCurrentUser()
		log.Logger().Infof("defaulting the user name to %s so we can create a PullRequest", pro.Username)
	}
	*/
	prDetails := &scm.PullRequest{}

	pr, err := pro.Create(devGitURL, "", prDetails, true)
	if err != nil {
		return errors.Wrapf(err, "failed to create Pull Request on the development environment git repository %s", devGitURL)
	}
	prURL := ""
	if pr != nil {
		prURL = pr.Link
		if o.WaitForSourceRepositoryPullRequest {
			err = o.waitForSourceRepositoryPullRequest(pr, devGitURL)
			if err != nil {
				return errors.Wrapf(err, "failed to wait for the Pull Request %s to merge", prURL)
			}
		}
	}
	o.GetReporter().CreatedDevRepoPullRequest(prURL, devGitURL)
	return nil
}
