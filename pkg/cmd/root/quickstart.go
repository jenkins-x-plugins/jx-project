package root

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/v3/pkg/files"
	"github.com/jenkins-x/jx-helpers/v3/pkg/gitclient"
	"github.com/jenkins-x/jx-helpers/v3/pkg/gitclient/giturl"
	"github.com/jenkins-x/jx-helpers/v3/pkg/options"
	"github.com/jenkins-x/jx-project/pkg/cmd/common"
	"github.com/jenkins-x/jx-project/pkg/cmd/importcmd"
	"github.com/jenkins-x/jx-project/pkg/quickstarts"
	"github.com/pkg/errors"

	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/spf13/cobra"

	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/templates"
)

var (
	createQuickstartLong = templates.LongDesc(`
		Create a new project from a sample/starter (found in https://github.com/jenkins-x-quickstarts)

		This will create a new project for you from the selected template.
		It will exclude any work-in-progress repos (containing the "WIP-" pattern)

		For more documentation see: [https://jenkins-x.io/developing/create-quickstart/](https://jenkins-x.io/developing/create-quickstart/)
`)

	createQuickstartExample = templates.Examples(`
		# create a new quickstart
		%s quickstart

		# creates a quickstart filtering on http based ones
		%s quickstart -f http
	`)
)

// CreateQuickstartOptions the options for the create quickstart command
type CreateQuickstartOptions struct {
	Options

	GitHubOrganisations []string
	Filter              quickstarts.QuickstartFilter
	GitHost             string
	IgnoreTeam          bool
}

// NewCmdCreateQuickstart creates a command object for the "create" command
func NewCmdCreateQuickstart() (*cobra.Command, *CreateQuickstartOptions) {
	o := &CreateQuickstartOptions{}

	cmd := &cobra.Command{
		Use:     "quickstart",
		Short:   "Create a new app from a Quickstart and import the generated code into Git and Jenkins for CI/CD",
		Long:    createQuickstartLong,
		Example: fmt.Sprintf(createQuickstartExample, common.BinaryName, common.BinaryName),
		Aliases: []string{"arch"},
		Run: func(cmd *cobra.Command, args []string) {
			o.Args = args
			err := o.Run()
			helper.CheckErr(err)
		},
	}
	o.addCreateAppFlags(cmd)

	cmd.Flags().StringArrayVarP(&o.GitHubOrganisations, "organisations", "g", []string{}, "The GitHub organisations to query for quickstarts")
	cmd.Flags().StringArrayVarP(&o.Filter.Tags, "tag", "t", []string{}, "The tags on the quickstarts to filter")
	cmd.Flags().StringVarP(&o.Filter.Owner, "owner", "", "", "The owner to filter on")
	cmd.Flags().StringVarP(&o.Filter.Language, "language", "l", "", "The language to filter on")
	cmd.Flags().StringVarP(&o.Filter.Framework, "framework", "", "", "The framework to filter on")
	cmd.Flags().StringVarP(&o.GitHost, "git-host", "", "", "The Git server host if not using GitHub when pushing created project")
	cmd.Flags().StringVarP(&o.Filter.Text, "filter", "f", "", "The text filter")
	cmd.Flags().StringVarP(&o.Filter.ProjectName, "project-name", "p", "", "The project name (for use with -b batch mode)")
	cmd.Flags().BoolVarP(&o.Filter.AllowML, "machine-learning", "", false, "Allow machine-learning quickstarts in results")
	return cmd, o
}

// Run implements the generic Create command
func (o *CreateQuickstartOptions) Run() error {
	err := o.Validate()
	if err != nil {
		return errors.Wrapf(err, "failed to validate options")
	}

	devEnvGitURL := o.DevEnv.Spec.Source.URL
	if devEnvGitURL == "" {
		return errors.Errorf("no spec.source.url for dev environment so cannot clone the version stream")
	}
	devEnvCloneDir, err := gitclient.CloneToDir(o.Git(), devEnvGitURL, "")
	if err != nil {
		return errors.Wrapf(err, "failed to clone dev environment git repository %s", devEnvGitURL)
	}

	versionStreamDir := filepath.Join(devEnvCloneDir, "versionStream")
	exists, err := files.DirExists(versionStreamDir)
	if err != nil {
		return errors.Wrapf(err, "failed to check if dir exists %s", versionStreamDir)
	}
	if !exists {
		return errors.Errorf("the dev Environment git repository %s does not include a versionStream directory", devEnvGitURL)
	}

	qo := &quickstarts.Options{
		VersionsDir: versionStreamDir,
		Namespace:   o.Namespace,
		CurrentUser: "",
		JXClient:    o.JXClient,
		ScmClient:   o.ScmFactory.ScmClient,
	}
	model, err := qo.LoadQuickStartsModel(o.GitHubOrganisations, o.IgnoreTeam)
	if err != nil {
		return fmt.Errorf("failed to load quickstarts: %s", err)
	}

	err = o.DefaultsFromTeamSettings()
	if err != nil {
		return err
	}

	q, err := model.CreateSurvey(&o.Filter, o.BatchMode, o.Input)
	if err != nil {
		return err
	}
	return o.CreateQuickStart(q)
}

// CreateQuickStart helper method to create a quickstart from a quickstart resource
func (o *CreateQuickstartOptions) CreateQuickStart(q *quickstarts.QuickstartForm) error {
	if q == nil {
		return fmt.Errorf("no quickstart chosen")
	}

	var details *importcmd.CreateRepoData
	o.GitRepositoryOptions.Namespace = o.ImportOptions.Organisation
	o.GitRepositoryOptions.Name = o.ImportOptions.Repository
	repoName := o.GitRepositoryOptions.Name
	if !o.BatchMode {
		var err error
		details, err = o.GetGitRepositoryDetails()
		if err != nil {
			return err
		}
		if details.RepoName != "" {
			repoName = details.RepoName
		}
		o.Filter.ProjectName = repoName
		if repoName == "" {
			return fmt.Errorf("No project name")
		}
		q.Name = repoName
	} else {
		q.Name = o.Filter.ProjectName
		if q.Name == "" {
			q.Name = repoName
		}
		if q.Name == "" {
			q.Name = o.Filter.Text
		}
		if q.Name == "" {
			return options.MissingOption("project-name")
		}

	}

	/* TODO
	githubAppMode, err := o.IsGitHubAppMode()
	if err != nil {
		return err
	}

	if githubAppMode {
		githubApp := &github.GithubApp{
			Factory: o.GetFactory(),
		}

		owner := o.GitRepositoryOptions.Owner
		repoName := o.GitRepositoryOptions.RepoName
		if details != nil {
			owner = details.Organisation
			repoName = details.RepoName
		}
		installed, err := githubApp.Install(owner, repoName, o.GetIOFileHandles(), true)
		if err != nil {
			return err
		}
		o.GithubAppInstalled = installed
	}
	*/

	currentUser, err := o.ScmFactory.GetUsername()
	if err != nil {
		return errors.Wrapf(err, "failed to get the current git user")
	}

	// Prevent accidental attempts to use ML Project Sets in create quickstart
	gitToken := o.ScmFactory.GitToken

	// lets not pass in a token if we are not using github
	if !strings.HasPrefix(o.ScmFactory.GitServerURL, giturl.GitHubURL) {
		gitToken = ""
	}
	if isMLProjectSet(q.Quickstart, currentUser, gitToken) {
		return fmt.Errorf("you have tried to select a machine-learning quickstart projectset please try again using jx create mlquickstart instead")
	}
	dir := o.OutDir
	if dir == "" {
		dir, err = os.Getwd()
		if err != nil {
			return err
		}
	}
	genDir, err := o.createQuickstart(q, dir, currentUser, gitToken)
	if err != nil {
		return err
	}

	// if there is a charts folder named after the app name, lets rename it to the generated app name
	folder := ""
	if q.Quickstart != nil {
		folder = q.Quickstart.Name
	}
	idx := strings.LastIndex(folder, "/")
	if idx > 0 {
		folder = folder[idx+1:]
	}
	if folder != "" {
		chartsDir := filepath.Join(genDir, "charts", folder)
		exists, err := files.FileExists(chartsDir)
		if err != nil {
			return err
		}
		if exists {
			o.PostDraftPackCallback = func() error {
				_, appName := filepath.Split(genDir)
				appChartDir := filepath.Join(genDir, "charts", appName)
				err := files.CopyDirOverwrite(chartsDir, appChartDir)
				if err != nil {
					return err
				}
				err = os.RemoveAll(chartsDir)
				if err != nil {
					return err
				}
				return gitclient.Remove(o.Git(), genDir, filepath.Join("charts", folder))
			}
		}
	}
	o.GetReporter().CreatedProject(genDir)

	o.Options.ImportOptions.ScmFactory.ScmClient = o.ScmFactory.ScmClient

	if details != nil {
		o.ConfigureImportOptions(details)
	}

	return o.ImportCreatedProject(genDir)
}

func (o *CreateQuickstartOptions) createQuickstart(f *quickstarts.QuickstartForm, dir, username, token string) (string, error) {
	q := f.Quickstart
	answer := filepath.Join(dir, f.Name)
	u := q.DownloadZipURL
	if u == "" {
		return answer, fmt.Errorf("quickstart %s does not have a download zip URL", q.ID)
	}
	client := http.Client{}

	req, err := http.NewRequest(http.MethodGet, u, strings.NewReader(""))
	if err != nil {
		return answer, err
	}

	if token != "" && username != "" {
		log.Logger().Debugf("Downloading Quickstart source zip from %s with basic auth for user: %s", u, username)
		req.SetBasicAuth(username, token)
	}
	res, err := client.Do(req)
	if err != nil {
		return answer, err
	}
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return answer, err
	}

	zipFile := filepath.Join(dir, "source.zip")
	err = ioutil.WriteFile(zipFile, body, files.DefaultFileWritePermissions)
	if err != nil {
		return answer, fmt.Errorf("failed to download file %s due to %s", zipFile, err)
	}
	tmpDir, err := ioutil.TempDir("", "jx-source-")
	if err != nil {
		return answer, fmt.Errorf("failed to create temporary directory: %s", err)
	}
	err = files.Unzip(zipFile, tmpDir)
	if err != nil {
		return answer, fmt.Errorf("failed to unzip new project file %s due to %s", zipFile, err)
	}
	err = os.Remove(zipFile)
	if err != nil {
		return answer, err
	}
	tmpDir, err = findFirstDirectory(tmpDir)
	if err != nil {
		return answer, fmt.Errorf("failed to find a directory inside the source download: %s", err)
	}
	err = files.RenameDir(tmpDir, answer, false)
	if err != nil {
		return answer, fmt.Errorf("failed to rename temp dir %s to %s: %s", tmpDir, answer, err)
	}
	o.GetReporter().GeneratedQuickStartAt(answer)
	return answer, nil
}

func findFirstDirectory(dir string) (string, error) {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return dir, err
	}
	for _, f := range files {
		if f.IsDir() {
			return filepath.Join(dir, f.Name()), nil
		}
	}
	return "", fmt.Errorf("no child directory found in %s", dir)
}

func isMLProjectSet(q *quickstarts.Quickstart, username, token string) bool {
	if !strings.HasPrefix(q.Name, "ML-") {
		return false
	}

	client := http.Client{}

	// Look at https://raw.githubusercontent.com/:owner/:repo/master/projectset
	u := "https://raw.githubusercontent.com/" + q.Owner + "/" + q.Name + "/master/projectset"

	req, err := http.NewRequest(http.MethodGet, u, strings.NewReader(""))
	if err != nil {
		log.Logger().Warnf("Problem creating request %s: %s ", u, err)
	}
	if token != "" && username != "" {
		log.Logger().Debugf("Trying to pull projectset file from %s with basic auth for user: %s", u, username)
		req.SetBasicAuth(username, token)
	}
	res, err := client.Do(req)
	if err != nil {
		return false
	}
	bodybytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Logger().Warnf("Problem parsing response body from %s: %s ", u, err)
		return false
	}
	body := string(bodybytes)
	return strings.Contains(body, "Tail")
}
