package importcmd

import (
	"path/filepath"

	jxcore "github.com/jenkins-x/jx-api/v4/pkg/apis/core/v4beta1"
	v1 "github.com/jenkins-x/jx-api/v4/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx-helpers/v3/pkg/files"
	"github.com/jenkins-x/jx-helpers/v3/pkg/gitclient/giturl"
	"github.com/jenkins-x/jx-helpers/v3/pkg/kube/naming"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/pkg/errors"
)

// IsRemoteClusterGitRepository returns true if we have detected a GitOps repository for Jenkins X 3.x
func IsRemoteClusterGitRepository(dir string) (bool, error) {
	fileNames := []string{
		filepath.Join(dir, ".lighthouse", "jenkins-x", "triggers.yaml"),
		filepath.Join(dir, "versionStream", "git-operator", "job.yaml"),
	}

	for _, f := range fileNames {
		exists, err := files.FileExists(f)
		if err != nil {
			return false, errors.Wrapf(err, "failed to check if file exists %s", f)
		}
		if !exists {
			return false, nil
		}
	}
	return true, nil
}

// allows any extra changes to be proposed to the dev environment pull request if needed
// e.g. if a new environment git repository is imported we should ensure we have an Environment created for the new environment
func (o *ImportOptions) modifyDevEnvironmentSource(importDir, promoteDir string, gitInfo *giturl.GitRepository, gitURL, gitKind, envName string, envStrategy v1.PromotionStrategyType) (bool, error) {
	log.Logger().Debugf("checking if the new repository is an Environment: %s", gitURL)

	gitops, err := IsRemoteClusterGitRepository(importDir)
	if err != nil {
		return false, errors.Wrapf(err, "failed to detect gitops repository for repo %s", gitURL)
	}
	if !gitops {
		return false, nil
	}

	requirementsResource, requirementsFileName, err := jxcore.LoadRequirementsConfig(promoteDir, false)
	if err != nil {
		return true, errors.Wrapf(err, "failed to load requirements file in repository %s", gitURL)
	}
	requirements := &requirementsResource.Spec
	if requirements != nil && requirementsFileName != "" {
		// lets make sure we have an environment for this  environment
		repoOwner := gitInfo.Organisation
		repoName := gitInfo.Name
		for k := range requirements.Environments {
			e := requirements.Environments[k]
			if e.Repository == repoName && e.Owner == repoOwner {
				log.Logger().Infof("the dev repository already has the gitops environment repository %s configured", gitURL)
				return true, nil
			}
		}

		// lets add a new environment

		if envName == "" {
			envName = naming.ToValidName(repoName)
		}
		requirements.Environments = append(requirements.Environments, jxcore.EnvironmentConfig{
			Key:               envName,
			Owner:             repoOwner,
			Repository:        repoName,
			GitServer:         gitInfo.HostURL(),
			GitKind:           gitKind,
			RemoteCluster:     true,
			PromotionStrategy: envStrategy,
		})
		err = requirementsResource.SaveConfig(requirementsFileName)
		if err != nil {
			return true, errors.Wrapf(err, "failed to save %s", requirementsFileName)
		}
	}
	return true, nil
}
