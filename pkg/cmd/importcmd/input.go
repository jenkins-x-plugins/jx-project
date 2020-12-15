package importcmd

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/jx-helpers/v3/pkg/scmhelpers"
	"github.com/jenkins-x/jx-helpers/v3/pkg/termcolor"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/pkg/errors"
)

type CreateRepoData struct {
	Organisation string
	RepoName     string
	FullName     string
	Private      bool
}

// PickOwner picks a git owner
func (o *ImportOptions) PickOwner(userName string) (string, error) {
	if userName == "" {
		userName = o.getCurrentUser()
	}
	names, err := o.getOwners(userName)
	if err != nil {
		return "", errors.Wrapf(err, "failed to find the organisations")
	}
	name, err := o.Input.PickNameWithDefault(names, "git owner:", userName, "pick the git owner (organisation or user) to create the repository")
	if err != nil {
		return name, errors.Wrapf(err, "failed to pick the owner")
	}
	return name, nil
}

// PickRepoName picks the repository name
func (o *ImportOptions) PickRepoName(owner string, defaultName string, allowExistingRepo bool) (string, error) {
	help := fmt.Sprintf("enter the name of the git repository to create within the %s owner", owner)

	validator := func(val interface{}) error {
		str, ok := val.(string)
		if !ok {
			return fmt.Errorf("Expected string value")
		}
		if strings.TrimSpace(str) == "" {
			return fmt.Errorf("Repository name is required")
		}
		if allowExistingRepo {
			return nil
		}
		return o.ValidateRepositoryName(owner, str)
	}

	name, err := o.Input.PickValidValue("git repository name:", defaultName, validator, help)
	if err != nil {
		return "", errors.Wrapf(err, "failed to choose the git repository")
	}
	return name, nil

}

// GetOrganizations gets the organisation
func (o *ImportOptions) getOwners(userName string) ([]string, error) {
	var names []string
	// include the username as a pseudo organization
	if userName != "" {
		names = append(names, userName)
	}

	ctx := context.Background()
	orgs, _, err := o.ScmFactory.ScmClient.Organizations.List(ctx, scm.ListOptions{
		Size: 500,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to list git organisations for user %s. Please update '$HOME/git/credentials' file with format 'https://<Username>:<Personal Access Token>@github.com'", userName)
	}
	for _, org := range orgs {
		names = append(names, org.Name)
	}
	sort.Strings(names)
	return names, nil
}

func (o *ImportOptions) PickNewOrExistingGitRepository() (*CreateRepoData, error) {
	if o.ScmFactory.GitToken == "" {
		return nil, errors.Errorf("TODO: please generate a personal access token")
	}

	/* TODO

	config := authConfigSvc.Config()

	var err error
	if server == nil {
		if repoOptions.ServerURL != "" {
			server = config.GetOrCreateServer(repoOptions.ServerURL)
		} else {
			if batchMode {
				if len(config.Servers) == 0 {
					return nil, fmt.Errorf("No Git servers are configured!")
				}
				// lets assume the first for now
				server = config.Servers[0]
				currentServer := config.CurrentServer
				if currentServer != "" {
					for _, s := range config.Servers {
						if s.Name == currentServer || s.URL == currentServer {
							server = s
							break
						}
					}
				}
			} else {
				server, err = config.PickServer("Which Git service?", batchMode, handles)
				if err != nil {
					return nil, err
				}
			}
			repoOptions.ServerURL = server.URL
		}
	}

	log.Logger().Infof("Using Git provider %s", termcolor.ColorInfo(server.Description()))
	url := server.URL

	if userAuth == nil {
		if repoOptions.Username != "" {
			userAuth = config.GetOrCreateUserAuth(url, repoOptions.Username)
			log.Logger().Infof(util.QuestionAnswer("Using Git user name", repoOptions.Username))
		} else {
			if batchMode {
				if len(server.Users) == 0 {
					return nil, fmt.Errorf("Server %s has no user auths defined", url)
				}
				var ua *auth.UserAuth
				if server.CurrentUser != "" {
					ua = config.FindUserAuth(url, server.CurrentUser)
				}
				if ua == nil {
					ua = server.Users[0]
				}
				userAuth = ua
			} else {
				userAuth, err = config.PickServerUserAuth(server, "Git user name?", batchMode, "", handles)
				if err != nil {
					return nil, err
				}
			}
		}
	}

	if userAuth.IsInvalid() && repoOptions.ApiToken != "" {
		userAuth.ApiToken = repoOptions.ApiToken
	}

	if userAuth.IsInvalid() {
		f := func(username string) error {
			git.PrintCreateRepositoryGenerateAccessToken(server, username, handles.Out)
			return nil
		}

		// TODO could we guess this based on the users ~/.git for github?
		defaultUserName := ""
		err = config.EditUserAuth(server.Label(), userAuth, defaultUserName, true, batchMode, f, handles)
		if err != nil {
			return nil, err
		}

		// TODO lets verify the auth works

		err = authConfigSvc.SaveUserAuth(url, userAuth)
		if err != nil {
			return nil, fmt.Errorf("Failed to store git auth configuration %s", err)
		}
		if userAuth.IsInvalid() {
			return nil, fmt.Errorf("You did not properly define the user authentication")
		}
	}

	gitUsername := userAuth.Username
	log.Logger().Debugf("About to create repository %s on server %s with user %s", termcolor.ColorInfo(defaultRepoName), termcolor.ColorInfo(url), termcolor.ColorInfo(gitUsername))

	provider, err := CreateProvider(server, userAuth, git)
	if err != nil {
		return nil, err
	}
	*/

	var err error
	repoOptions := &o.GitRepositoryOptions

	gitUsername, err := o.ScmFactory.GetUsername()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get current username")
	}
	owner := repoOptions.Namespace
	if owner == "" {
		owner, err = o.GetOwner(gitUsername)
		if err != nil {
			return nil, err
		}
	} else {
		log.Logger().Infof(QuestionAnswer("Using organisation", owner))
	}

	defaultRepoName := ""
	repoName := repoOptions.Name
	if repoName == "" {
		repoName, err = o.PickRepoName(owner, defaultRepoName, false)
		if err != nil {
			return nil, err
		}
	} else {
		if !o.IgnoreExistingRepository {
			err := o.ValidateRepositoryName(owner, repoName)
			if err != nil {
				return nil, err
			}
			log.Logger().Infof(QuestionAnswer("Using repository", repoName))
		}
	}

	fullName := scm.Join(owner, repoName)
	log.Logger().Infof("Creating repository %s", termcolor.ColorInfo(fullName))

	return &CreateRepoData{
		Organisation: owner,
		RepoName:     repoName,
		FullName:     fullName,
		Private:      repoOptions.Private,
	}, err
}

// ValidateRepositoryName validates the repository does not exist
func (o *ImportOptions) ValidateRepositoryName(owner string, name string) error {
	fullName := scm.Join(owner, name)
	ctx := context.Background()
	_, _, err := o.ScmFactory.ScmClient.Repositories.Find(ctx, fullName)
	if err == nil {
		return errors.Errorf("repository %s already exists", fullName)
	}
	if scmhelpers.IsScmNotFound(err) {
		return nil
	}
	return errors.Wrapf(err, "failed to check if repository %s exists", fullName)
}

// QuestionAnswer returns strings like Cobra question/answers for default cli options
func QuestionAnswer(question string, answer string) string {
	return fmt.Sprintf("%s %s: %s", termcolor.ColorBold(termcolor.ColorInfo("?")), termcolor.ColorBold(question), termcolor.ColorAnswer(answer))
}

/* TODO
func (o *ImportOptions) GetRepoName(batchMode, allowExistingRepo bool, provider GitProvider, defaultRepoName, owner string, handles util.IOFileHandles) (string, error) {
	surveyOpts := survey.WithStdio(handles.In, handles.Out, handles.Err)
	repoName := ""
	if batchMode {
		repoName = defaultRepoName
		if repoName == "" {
			repoName = "dummy"
		}
	} else {
		prompt := &survey.Input{
			Message: "Enter the new repository name: ",
			Default: defaultRepoName,
		}
		validator := func(val interface{}) error {
			str, ok := val.(string)
			if !ok {
				return fmt.Errorf("Expected string value")
			}
			if strings.TrimSpace(str) == "" {
				return fmt.Errorf("Repository name is required")
			}
			if allowExistingRepo {
				return nil
			}
			return provider.ValidateRepositoryName(owner, str)
		}
		err := survey.AskOne(prompt, &repoName, validator, surveyOpts)
		if err != nil {
			return "", err
		}
		if repoName == "" {
			return "", fmt.Errorf("No repository name specified")
		}
	}
	return repoName, nil
}

*/

func (o *ImportOptions) GetOwner(gitUsername string) (string, error) {
	owner := ""
	if o.BatchMode {
		owner = gitUsername
	} else {
		var err error
		owner, err = o.PickOwner(gitUsername)
		if err != nil {
			return "", err
		}
		if owner == "" {
			owner = gitUsername
		}
	}
	return owner, nil
}
