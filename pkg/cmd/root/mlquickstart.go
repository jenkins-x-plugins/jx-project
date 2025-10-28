package root

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jenkins-x-plugins/jx-project/pkg/cmd/importcmd"

	"github.com/jenkins-x-plugins/jx-project/pkg/quickstarts"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/templates"
	"github.com/jenkins-x/jx-helpers/v3/pkg/files"
	"github.com/jenkins-x/jx-helpers/v3/pkg/gitclient"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/pkg/errors"

	"github.com/spf13/cobra"
)

var (
	createMLQuickstartLong = templates.LongDesc(`
		Create a new machine learning project from a sample/starter (found in https://github.com/machine-learning-quickstarts)

		This will create two new projects for you from the selected template. One for training and one for deploying a model as a service.
		It will exclude any work-in-progress repos (containing the "WIP-" pattern)

		For more documentation see: https://jenkins-x.io/v3/mlops/

` + helper.SeeAlsoText("jx project"))

	createMLQuickstartExample = templates.Examples(`
		jx project mlquickstart

		jx project mlquickstart -f pytorch
	`)
)

// CreateMLQuickstartOptions the options for the create quickstart command
type CreateMLQuickstartOptions struct {
	Options

	GitHubOrganisations []string
	Filter              quickstarts.QuickstartFilter
	GitHost             string
	QuickstartAuth      string
}

type projectset struct {
	Repo string
	Tail string
}

// NewCmdCreateMLQuickstart creates a command object for the "project" command
func NewCmdCreateMLQuickstart() (*cobra.Command, *CreateMLQuickstartOptions) {
	options := &CreateMLQuickstartOptions{}

	cmd := &cobra.Command{
		Use:     "mlquickstart",
		Short:   "Create a new machine learning app from a set of quickstarts and import the generated code into Git and Jenkins for CI/CD",
		Long:    createMLQuickstartLong,
		Example: createMLQuickstartExample,
		Aliases: []string{"arch"},
		Run: func(_ *cobra.Command, args []string) {
			options.Args = args
			err := options.Run()
			helper.CheckErr(err)
		},
	}
	options.addCreateAppFlags(cmd)

	cmd.Flags().StringArrayVarP(&options.GitHubOrganisations, "organisations", "g", []string{}, "The GitHub organisations to query for quickstarts")
	cmd.Flags().StringArrayVarP(&options.Filter.Tags, "tag", "t", []string{}, "The tags on the quickstarts to filter")
	cmd.Flags().StringVarP(&options.QuickstartAuth, "quickstart-auth", "", "", "The auth mechanism used to authenticate with the git token to download the quickstarts. If not specified defaults to Basic but could be Bearer for bearer token auth")
	cmd.Flags().StringVarP(&options.Filter.Owner, "owner", "", "", "The owner to filter on")
	cmd.Flags().StringVarP(&options.Filter.Language, "language", "l", "", "The language to filter on")
	cmd.Flags().StringVarP(&options.Filter.Framework, "framework", "", "", "The framework to filter on")
	cmd.Flags().StringVarP(&options.GitHost, "git-host", "", "", "The Git server host if not using GitHub when pushing created project")
	cmd.Flags().StringVarP(&options.Filter.Text, "filter", "f", "", "The text filter")
	cmd.Flags().StringVarP(&options.Filter.ProjectName, "project-name", "p", "", "The project name (for use with -b batch mode)")
	return cmd, options
}

// Run implements the generic Create command
func (o *CreateMLQuickstartOptions) Run() error {
	log.Logger().Debugf("Running CreateMLQuickstart...\n")

	// Force ML quickstarts in filter
	o.Filter.AllowML = true

	err := o.Validate()
	if err != nil {
		return errors.Wrapf(err, "failed to validate options")
	}

	interactive := true
	if o.BatchMode {
		interactive = false
		log.Logger().Debugf("In batch mode.\n")
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
	log.Logger().Debugf("versionStream: %s\n", versionStreamDir)

	qo := &quickstarts.Options{
		VersionsDir: versionStreamDir,
		Namespace:   o.Namespace,
		CurrentUser: "",
		JXClient:    o.JXClient,
		ScmClient:   o.ScmFactory.ScmClient,
	}

	var details *importcmd.CreateRepoData
	log.Logger().Debugf("Ask for details of where to put the projects")
	if interactive {
		log.Logger().Info("Where would you like to put this project?\n")
		log.Logger().Info("The name you enter will be used as the prefix when creating your ML repos. For example: mymlproject-service and mymlproject-training.\n")
		details, err = o.GetGitRepositoryDetails()
		if err != nil {
			return err
		}

		o.Filter.ProjectName = details.RepoName
	}

	log.Logger().Debugf("About to LoadMLProjectSetsModel...\n")
	model, err := qo.LoadMLProjectSetsModel(o.GitHubOrganisations)
	if err != nil {
		return fmt.Errorf("failed to load mlprojectsets: %s", err)
	}

	err = o.DefaultsFromTeamSettings()
	if err != nil {
		return err
	}

	var q *quickstarts.QuickstartForm
	if o.BatchMode {
		q, err = pickMLProject(model, &o.Filter)
	} else {
		log.Logger().Debugf("Creating survey...\n")
		log.Logger().Debugf("Filter: %v\n", o.Filter)
		q, err = model.CreateSurvey(&o.Filter, o.BatchMode, o.Input)
	}

	if err != nil {
		return err
	}
	if q == nil {
		return fmt.Errorf("no quickstart chosen")
	}

	dir := o.OutDir
	if dir == "" {
		_, err = os.Getwd()
		if err != nil {
			return err
		}
	}
	log.Logger().Debugf("Transferring options...\n")
	w := &CreateQuickstartOptions{}
	w.Options = o.Options
	w.GitHubOrganisations = o.GitHubOrganisations
	w.Filter = o.Filter
	w.Filter.Text = q.Quickstart.Name
	w.QuickstartAuth = o.QuickstartAuth
	w.GitHost = o.GitHost

	// Switch to BatchMode from here on
	o.BatchMode = true
	prefix := o.Filter.ProjectName
	projectImportOptions := o.ImportOptions

	// Check to see if the selection is a project set
	ps, err := o.getMLProjectSet(q.Quickstart)

	var e error
	if err == nil {
		// We have a projectset so create all the associated quickstarts
		for _, project := range ps {
			w.ImportOptions = projectImportOptions // Reset the options each time as they are modified by Import (DraftPack)
			if interactive {
				log.Logger().Debugf("Setting Quickstart from surveys.\n")
				w.ImportOptions.Organisation = details.Organisation
				w.GitRepositoryOptions = o.GitRepositoryOptions
			}
			w.Filter.Text = project.Repo
			w.Filter.ProjectName = prefix + project.Tail
			w.ImportOptions.Repository = w.Filter.ProjectName // For Draft
			w.Filter.Language = ""
			log.Logger().Debugf("Invoking CreateQuickstart for %s...\n", project.Repo)

			e = w.Run()

			if e != nil {
				return e
			}
		}
	} else {
		// Must be a conventional quickstart. This path shouldn't be reachable.
		log.Logger().Debugf("Invoking CreateQuickstart...\n")
		return w.Run()
	}
	log.Logger().Info("")
	log.Logger().Infof("Once your training script completes, remember to check the PR on %s/%s/%s%s/pulls to merge the trained model into your service.\n", o.ImportOptions.ScmFactory.GitServerURL, details.Organisation, prefix, ps[0].Tail)
	return e

}

func (o *CreateMLQuickstartOptions) getMLProjectSet(q *quickstarts.Quickstart) ([]projectset, error) {
	var ps []projectset

	// Look at https://raw.githubusercontent.com/:owner/:repo/master/projectset
	client := http.Client{}
	u := "https://raw.githubusercontent.com/" + q.Owner + "/" + q.Name + "/master/projectset"

	req, err := http.NewRequest(http.MethodGet, u, strings.NewReader(""))
	if err != nil {
		log.Logger().Debugf("Projectset not found because %+#v\n", err)
		return nil, err
	}
	// TODO: Handle private repos
	// userAuth := q.GitProvider.UserAuth()
	// token := userAuth.ApiToken
	// username := userAuth.Username
	// if token != "" && username != "" {
	// 	log.Logger().Debugf("Downloading projectset from %s with basic auth for user: %s\n", u, username)
	// 	req.SetBasicAuth(username, token)
	// }
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(body, &ps)
	return ps, err
}

// PickMLProject picks a mlquickstart project set from filtered results
func pickMLProject(model *quickstarts.QuickstartModel, filter *quickstarts.QuickstartFilter) (*quickstarts.QuickstartForm, error) {
	mlquickstarts := model.Filter(filter)
	names := []string{}
	m := map[string]*quickstarts.Quickstart{}
	for _, qs := range mlquickstarts {
		name := qs.SurveyName()
		m[name] = qs
		names = append(names, name)
	}
	sort.Strings(names)

	if len(names) == 0 {
		return nil, fmt.Errorf("no quickstarts match filter")
	}
	answer := ""
	// Pick the first option as this is the project set
	answer = names[0]
	if answer == "" {
		return nil, fmt.Errorf("no quickstart chosen")
	}
	q := m[answer]
	if q == nil {
		return nil, fmt.Errorf("could not find chosen quickstart for %s", answer)
	}
	form := &quickstarts.QuickstartForm{
		Quickstart: q,
		Name:       q.Name,
	}
	return form, nil
}
