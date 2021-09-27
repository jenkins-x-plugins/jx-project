package importcmd

import (
	"context"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/jx-helpers/v3/pkg/boot"
	"github.com/jenkins-x/jx-helpers/v3/pkg/scmhelpers"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/pkg/errors"
)

func (o *ImportOptions) AddAndAcceptCollaborator(newRepository bool) error {
	ctx := context.Background()
	githubAppMode, err := o.IsGitHubAppMode()
	if err != nil {
		return err
	}
	if githubAppMode {
		return nil
	}

	userName := o.getCurrentUser()

	scmClient := o.ScmFactory.ScmClient
	if scmClient == nil {
		return errors.Errorf("no SCM client")
	}
	owner := o.Organisation
	if owner == "" {
		return errors.Errorf("no organisation")
	}
	repoName := o.Repository
	if repoName == "" {
		repoName = o.AppName
	}
	if repoName == "" {
		repoName = o.GitRepositoryOptions.Name
	}
	if repoName == "" {
		return errors.Errorf("no repository")
	}

	fullRepoName := repoName
	if owner != userName || o.ScmFactory.GitKind == "gitlab" {
		fullRepoName = scm.Join(owner, fullRepoName)
	}

	permission := "admin"
	pipelineUserName := o.PipelineUserName
	if !newRepository {
		// lets check if the pipeline user is already a collaborator
		collaborator, _, err := scmClient.Repositories.IsCollaborator(ctx, fullRepoName, pipelineUserName)
		if err != nil {
			return errors.Wrapf(err, "failed to check if %s is a collaborator on %s", pipelineUserName, fullRepoName)
		}

		if collaborator {
			return nil
		}
	}

	// If the user creating the repo is not the pipeline user, add the pipeline user as a contributor to the repo
	if pipelineUserName != "" && pipelineUserName != userName { // TODO: not sure why:  && o.ScmFactory.GitServerURL == o.PipelineServer {
		// Make the invitation
		alreadyMember := false
		_, alreadyMember, _, err = scmClient.Repositories.AddCollaborator(ctx, fullRepoName, pipelineUserName, permission)
		if alreadyMember {
			return nil
		}
		if err != nil {
			return errors.Wrapf(err, "failed to add %s as a collaborator to %s", pipelineUserName, fullRepoName)
		}

		if o.OperatorNamespace == "" {
			o.OperatorNamespace = boot.GitOperatorNamespace
		}
		if o.BootSecretName == "" {
			o.BootSecretName = boot.SecretName
		}
		bootSecret, err := boot.LoadBootSecret(o.KubeClient, o.OperatorNamespace, o.OperatorNamespace, o.BootSecretName, pipelineUserName)
		if err != nil {
			return errors.Wrapf(err, "failed to load the boot secret")
		}

		if bootSecret.Username == "" {
			bootSecret.Username = pipelineUserName
		}

		f := scmhelpers.Factory{
			GitKind:      o.ScmFactory.GitKind,
			GitServerURL: o.ScmFactory.GitServerURL,
			GitUsername:  bootSecret.Username,
			GitToken:     bootSecret.Password,
		}
		bootScmClient, err := f.Create()
		if err != nil {
			return errors.Wrapf(err, "failed to create SCM client for boot user %s on server %s", f.GitUsername, f.GitServerURL)
		}
		o.BootScmClient = bootScmClient

		if o.ScmFactory.GitKind == "gitea" {
			// AddCollaborator doesn't use invitations
			return nil
		}

		// Get all invitations for the pipeline user
		invites, _, err := bootScmClient.Users.ListInvitations(ctx)
		if err != nil {
			return errors.Wrapf(err, "failed to list invites")
		}
		for i := range invites {
			invite := invites[i]
			repository := invite.Repo
			if repository != nil && repository.Name == repoName {
				_, err = bootScmClient.Users.AcceptInvitation(ctx, invite.ID)
				if err != nil {
					log.Logger().Warnf("failed to accept invitation %v on repository %s: %s", invite.ID, invite.Repo.FullName, err.Error())
				}
				log.Logger().Infof("accepted invitation %v for repository %s", invite.ID, info(invite.Repo.FullName))
			}
		}
	}
	return nil
}
