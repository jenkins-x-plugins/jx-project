package importcmd

import (
	"context"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/jenkins-x/jx-helpers/v3/pkg/files"
	"github.com/jenkins-x/jx-helpers/v3/pkg/gitclient/giturl"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/pkg/errors"
)

// HasJenkinsfile returns  the ile name if there is a Jenkinsfile or empty string if there is not
func (o *ImportOptions) HasJenkinsfile() (string, error) {
	dir := o.Dir
	var err error

	jenkinsfile := jenkinsfileName
	if o.Jenkinsfile != "" {
		jenkinsfile = o.Jenkinsfile
	}
	if !filepath.IsAbs(jenkinsfile) {
		jenkinsfile = filepath.Join(dir, jenkinsfile)
	}
	exists, err := files.FileExists(jenkinsfile)
	if err != nil {
		return "", err
	}
	if !exists {
		return "", nil
	}
	return jenkinsfile, nil
}

// EvaluateBuildPack performs an evaluation of the build pack on the current source
func (o *ImportOptions) EvaluateBuildPack(devEnvCloneDir, jenkinsfile string) error {
	// TODO this is a workaround of this draft issue:
	// https://github.com/Azure/draft/issues/476
	var err error

	args := &InvokeDraftPack{
		Dir:             o.Dir,
		DevEnvCloneDir:  devEnvCloneDir,
		CustomDraftPack: o.Pack,
		Jenkinsfile:     jenkinsfile,
		InitialisedGit:  o.InitialisedGit,
	}
	o.Pack, err = o.InvokeDraftPack(args)
	if err != nil {
		return err
	}

	// lets rename the chart to be the same as our app name
	err = o.renameChartToMatchAppName()
	if err != nil {
		return err
	}

	/* TODO
	err = o.modifyDeployKind()
	if err != nil {
		return err
	}

	*/
	if o.PostDraftPackCallback != nil {
		err = o.PostDraftPackCallback()
		if err != nil {
			return err
		}
	}

	gitServerName := ""
	if o.gitInfo != nil {
		gitServerName = o.gitInfo.Host
	}
	gitServerURL := o.ScmFactory.GitServerURL
	if gitServerName == "" {
		if gitServerURL == "" {
			return errors.Errorf("no git server URL")
		}
		u, err := url.Parse(gitServerURL)
		if err != nil {
			return errors.Wrapf(err, "failed to parse git server URL %s", gitServerURL)
		}
		gitServerName = u.Host
	}
	if gitServerName == "" {
		return errors.Errorf("no git server name")
	}

	if o.Organisation == "" {
		o.Organisation, err = o.PickOwner("")
		if err != nil {
			return errors.Wrapf(err, "failed to pick a git owner")
		}
	}

	if o.AppName == "" {
		_, defaultRepoName := filepath.Split(o.Dir)

		o.AppName, err = o.PickRepoName(o.Organisation, defaultRepoName, false)
		if err != nil {
			return err
		}
	}

	dockerRegistryOrg := o.getDockerRegistryOrg()
	err = o.ReplacePlaceholders(gitServerName, dockerRegistryOrg)
	if err != nil {
		return err
	}

	// Create Prow owners file
	err = o.CreateProwOwnersFile()
	if err != nil {
		return err
	}
	err = o.CreateProwOwnersAliasesFile()
	if err != nil {
		return err
	}
	return nil
}

func (o *ImportOptions) getDockerRegistryOrg() string {
	dockerRegistryOrg := o.DockerRegistryOrg
	if dockerRegistryOrg == "" {
		dockerRegistryOrg = o.getOrganisationOrCurrentUser()
	}
	return strings.ToLower(dockerRegistryOrg)
}

func (o *ImportOptions) getOrganisationOrCurrentUser() string {
	org := o.GetOrganisation()
	if org == "" {
		org = o.getCurrentUser()
	}
	return org
}

func (o *ImportOptions) getCurrentUser() string {
	// walk through every file in the given dir and update the placeholders
	if o.ScmFactory.GitUsername == "" {
		if o.ScmFactory.ScmClient != nil {
			ctx := context.Background()
			user, _, err := o.ScmFactory.ScmClient.Users.Find(ctx)
			if err != nil {
				log.Logger().Warnf("failed to find current user in git %s", err.Error())
			} else {
				o.ScmFactory.GitUsername = user.Login
			}
		}
	}
	if o.ScmFactory.GitUsername == "" {
		log.Logger().Warn("No username defined for the current Git server!")
	}
	return o.ScmFactory.GitUsername
}

// GetOrganisation gets the organisation from the RepoURL (if in the github format of github.com/org/repo). It will
// do this in preference to the Organisation field (if set). If the repo URL does not implicitly specify an organisation
// then the Organisation specified in the options is used.
func (o *ImportOptions) GetOrganisation() string {
	org := ""
	if o.DiscoveredGitURL == "" {
		o.DiscoveredGitURL = o.RepoURL
	}
	gitInfo, err := giturl.ParseGitURL(o.DiscoveredGitURL)
	if err == nil && gitInfo.Organisation != "" {
		org = gitInfo.Organisation
		if o.Organisation != "" && org != o.Organisation {
			log.Logger().Warnf("organisation %s detected from URL %s. '--org %s' will be ignored", org, o.DiscoveredGitURL, o.Organisation)
		}
	} else {
		org = o.Organisation
	}
	return org
}
