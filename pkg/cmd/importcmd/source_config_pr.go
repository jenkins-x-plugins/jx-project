package importcmd

import (
	"fmt"
	"path/filepath"

	"github.com/jenkins-x-plugins/jx-gitops/pkg/cmd/repository/add"
	"github.com/jenkins-x-plugins/jx-promote/pkg/environments"
	"github.com/jenkins-x/go-scm/scm"
	v1 "github.com/jenkins-x/jx-api/v4/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx-helpers/v3/pkg/kube/jxenv"
	"github.com/jenkins-x/jx-helpers/v3/pkg/stringhelpers"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/pkg/errors"
)

func (o *ImportOptions) addSourceConfigPullRequest(gitURL, gitKind string) (bool, error) {
	remoteCluster := false
	if o.NoDevPullRequest {
		return remoteCluster, nil
	}
	devEnv, err := jxenv.GetDevEnvironment(o.JXClient, o.Namespace)
	if err != nil {
		return remoteCluster, errors.Wrapf(err, "failed to find the dev Environment")
	}

	log.Logger().Info("")
	log.Logger().Info("we are now going to create a Pull Request on the development cluster git repository to setup CI/CD via GitOps")
	log.Logger().Info("")

	safeGitURL := stringhelpers.SanitizeURL(gitURL)

	devGitURL := devEnv.Spec.Source.URL
	if devGitURL == "" {
		return remoteCluster, errors.Errorf("no git source URL for Environment %s", devEnv.Name)
	}

	// let's generate a PR
	if o.SchedulerName == "" {
		g := filepath.Join(o.Dir, ".lighthouse", "*", "triggers.yaml")
		matches, err := filepath.Glob(g)
		if err != nil {
			return remoteCluster, errors.Wrapf(err, "failed to evaluate glob %s", g)
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
		CommitTitle:       fmt.Sprintf("chore: import repository %s", safeGitURL),
		CommitMessage:     "this commit will trigger a pipeline to [generate the CI/CD configuration](https://jenkins-x.io/v3/about/how-it-works/#importing--creating-quickstarts) which will create a second commit on this Pull Request before it auto merges",
		ScmClient:         o.ScmFactory.ScmClient,
		BatchMode:         o.BatchMode,
		UseGitHubOAuth:    false,
		Fork:              false,
		// Labels:            []string{"env/dev"},
	}

	pro.Function = func() error {
		dir := pro.OutDir
		_, ao := add.NewCmdAddRepository()
		ao.Args = []string{safeGitURL}
		ao.Dir = dir
		ao.JXClient = o.JXClient
		ao.Namespace = o.Namespace
		ao.Scheduler = o.SchedulerName
		ao.Jenkins = o.Destination.Jenkins.Server
		err := ao.Run()
		if err != nil {
			return errors.Wrapf(err, "failed to add git URL %s to the source-config.yaml file", safeGitURL)
		}

		remoteCluster, err = o.modifyDevEnvironmentSource(o.Dir, dir, o.gitInfo, safeGitURL, gitKind, o.EnvName, v1.PromotionStrategyType(o.EnvStrategy))
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
	prDetails := &scm.PullRequest{
		Labels: []*scm.Label{
			{
				Name: "env/dev",
			},
		},
	}

	pr, err := pro.Create(devGitURL, "", prDetails, false)
	if err != nil {
		return remoteCluster, errors.Wrapf(err, "failed to create Pull Request on the development environment git repository %s", devGitURL)
	}
	prURL := ""
	if pr != nil {
		prURL = pr.Link
		if o.WaitForSourceRepositoryPullRequest {

			log.Logger().Info("")
			log.Logger().Info("we now need to wait for the Pull Request to merge so that CI/CD can be setup via GitOps")
			log.Logger().Info("")

			err = o.waitForSourceRepositoryPullRequest(pr)
			if err != nil {
				return remoteCluster, errors.Wrapf(err, "failed to wait for the Pull Request %s to merge", prURL)
			}
		}
	}
	o.GetReporter().CreatedDevRepoPullRequest(prURL, devGitURL)
	return remoteCluster, nil
}
