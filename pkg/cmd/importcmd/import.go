package importcmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/denormal/go-gitignore"
	"github.com/jenkins-x-labs/trigger-pipeline/pkg/jenkinsutil"
	"github.com/jenkins-x-labs/trigger-pipeline/pkg/jenkinsutil/factory"
	gojenkins "github.com/jenkins-x/golang-jenkins"
	"github.com/jenkins-x/jx-logging/pkg/log"
	"github.com/jenkins-x/jx-project/pkg/cmd/common"
	jenkinsio "github.com/jenkins-x/jx/v2/pkg/apis/jenkins.io"
	v1 "github.com/jenkins-x/jx/v2/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/v2/pkg/auth"
	"github.com/jenkins-x/jx/v2/pkg/cloud/amazon"
	"github.com/jenkins-x/jx/v2/pkg/cmd/edit"
	"github.com/jenkins-x/jx/v2/pkg/cmd/helper"
	"github.com/jenkins-x/jx/v2/pkg/cmd/opts"
	"github.com/jenkins-x/jx/v2/pkg/cmd/start"
	"github.com/jenkins-x/jx/v2/pkg/cmd/step/create/pr"
	"github.com/jenkins-x/jx/v2/pkg/cmd/templates"
	"github.com/jenkins-x/jx/v2/pkg/config"
	"github.com/jenkins-x/jx/v2/pkg/github"
	"github.com/jenkins-x/jx/v2/pkg/gits"
	"github.com/jenkins-x/jx/v2/pkg/jenkinsfile"
	"github.com/jenkins-x/jx/v2/pkg/jxfactory"
	"github.com/jenkins-x/jx/v2/pkg/kube"
	"github.com/jenkins-x/jx/v2/pkg/kube/naming"
	"github.com/jenkins-x/jx/v2/pkg/maven"
	"github.com/jenkins-x/jx/v2/pkg/prow"
	"github.com/jenkins-x/jx/v2/pkg/tekton/syntax"
	"github.com/jenkins-x/jx/v2/pkg/util"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1"
	gitcfg "gopkg.in/src-d/go-git.v4/config"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

// CallbackFn callback function
type CallbackFn func() error

// ImportOptions options struct for jwizard import
type ImportOptions struct {
	*opts.CommonOptions

	RepoURL                string
	Dir                    string
	Organisation           string
	Repository             string
	Credentials            string
	AppName                string
	SelectFilter           string
	Jenkinsfile            string
	BranchPattern          string
	ImportGitCommitMessage string
	Pack                   string
	DockerRegistryOrg      string
	DeployKind             string
	SchedulerName          string
	GitConfDir             string
	PipelineUserName       string
	PipelineServer         string
	ImportMode             string
	ServiceAccount         string
	DisableMaven           bool
	UseDefaultGit          bool
	GithubAppInstalled     bool
	GitHub                 bool
	DryRun                 bool
	SelectAll              bool
	DisableBuildPack       bool
	DisableWebhooks        bool
	DisableDotGitSearch    bool
	InitialisedGit         bool
	DeployOptions          v1.DeployOptions
	GitRepositoryOptions   gits.GitRepositoryOptions
	GitDetails             gits.CreateRepoData
	Jenkins                gojenkins.JenkinsClient
	GitServer              *auth.AuthServer
	GitUserAuth            *auth.UserAuth
	GitProvider            gits.GitProvider
	PostDraftPackCallback  CallbackFn
	JXFactory              jxfactory.Factory

	Destination          ImportDestination
	reporter             ImportReporter
	jenkinsClientFactory *jenkinsutil.ClientFactory
}

const (
	triggerPipelineBuildPack   = "trigger-jenkins"
	jenkinsfileRunnerBuildPack = "jenkinsfilerunner"
	jenkinsServerEnvVar        = "TRIGGER_JENKINS_SERVER"

	// TODO until `jx` can handle overrides of step images without having to copy/paste the command too we need to copy paste the command
	// from the build pack if we wish to override the image name
	defaultJenkinsfileRunnerCommand = "/app/bin/jenkinsfile-runner-launcher -w /app/jenkins -p /usr/share/jenkins/ref/plugins -f /workspace/source --runWorkspace /workspace/build"
)

var (
	importLong = templates.LongDesc(`
		Imports a local folder or Git repository into Jenkins X.

		If you specify no other options or arguments then the current directory is imported.
	    Or you can use '--dir' to specify a directory to import.

	    You can specify the git URL as an argument.
	    
		For more documentation see: [https://jenkins-x.io/docs/using-jx/creating/import/](https://jenkins-x.io/docs/using-jx/creating/import/)
	    
`)

	importExample = templates.Examples(`
		# Import the current folder
		%s import

		# Import a different folder
		%s import /foo/bar

		# Import a Git repository from a URL
		%s import --url https://github.com/jenkins-x/spring-boot-web-example.git

        # Select a number of repositories from a GitHub organisation
		%s import --github --org myname 

        # Import all repositories from a GitHub organisation selecting ones to not import
		%s import --github --org myname --all 

        # Import all repositories from a GitHub organisation which contain the text foo
		%s import --github --org myname --all --filter foo 
		`)

	deployKinds = []string{opts.DeployKindKnative, opts.DeployKindDefault}

	removeSourceRepositoryAnnotations = []string{"kubectl.kubernetes.io/last-applied-configuration", "jenkins.io/chart"}
)

// NewCmdImport the cobra command for jwizard import
func NewCmdImport(commonOpts *opts.CommonOptions) *cobra.Command {
	cmd, _ := NewCmdImportAndOptions(commonOpts)
	return cmd
}

// NewCmdImportAndOptions creates the cobra command for jwizard import and the options
func NewCmdImportAndOptions(commonOpts *opts.CommonOptions) (*cobra.Command, *ImportOptions) {
	options := &ImportOptions{
		CommonOptions: commonOpts,
	}
	cmd := &cobra.Command{
		Use:     "import",
		Short:   "Imports a local project or Git repository into Jenkins",
		Long:    importLong,
		Example: fmt.Sprintf(importExample, common.BinaryName, common.BinaryName, common.BinaryName, common.BinaryName, common.BinaryName, common.BinaryName),
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			helper.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&options.RepoURL, "url", "u", "", "The git clone URL to clone into the current directory and then import")
	cmd.Flags().BoolVarP(&options.GitHub, "github", "", false, "If you wish to pick the repositories from GitHub to import")
	cmd.Flags().BoolVarP(&options.SelectAll, "all", "", false, "If selecting projects to import from a Git provider this defaults to selecting them all")
	cmd.Flags().StringVarP(&options.SelectFilter, "filter", "", "", "If selecting projects to import from a Git provider this filters the list of repositories")
	options.AddImportFlags(cmd, false)
	options.Destination.Jenkins.JenkinsSelectorOptions.AddFlags(cmd)
	options.Cmd = cmd
	return cmd, options
}

func (o *ImportOptions) AddImportFlags(cmd *cobra.Command, createProject bool) {
	notCreateProject := func(text string) string {
		if createProject {
			return ""
		}
		return text
	}
	cmd.Flags().StringVarP(&o.Organisation, "org", "", "", "Specify the Git provider organisation to import the project into (if it is not already in one)")
	cmd.Flags().StringVarP(&o.Repository, "name", "", notCreateProject("n"), "Specify the Git repository name to import the project into (if it is not already in one)")
	cmd.Flags().StringVarP(&o.Credentials, "credentials", notCreateProject("c"), "", "The Jenkins credentials name used by the job")
	cmd.Flags().StringVarP(&o.Jenkinsfile, "jenkinsfile", notCreateProject("j"), "", "The name of the Jenkinsfile to use. If not specified then 'Jenkinsfile' will be used")
	cmd.Flags().BoolVarP(&o.DryRun, "dry-run", "", false, "Performs local changes to the repo but skips the import into Jenkins X")
	cmd.Flags().BoolVarP(&o.DisableBuildPack, "no-pack", "", false, "Disable trying to default a Dockerfile and Helm Chart from the build pack")
	cmd.Flags().StringVarP(&o.ImportGitCommitMessage, "import-commit-message", "", "", "Specifies the initial commit message used when importing the project")
	cmd.Flags().StringVarP(&o.BranchPattern, "branches", "", "", "The branch pattern for branches to trigger CI/CD pipelines on")
	cmd.Flags().StringVarP(&o.Pack, "pack", "", "", "The name of the build pack to use. If none is specified it will be chosen based on matching the source code languages")
	cmd.Flags().StringVarP(&o.SchedulerName, "scheduler", "", "", "The name of the Scheduler configuration to use for ChatOps when using Prow")
	cmd.Flags().StringVarP(&o.DockerRegistryOrg, "docker-registry-org", "", "", "The name of the docker registry organisation to use. If not specified then the Git provider organisation will be used")
	cmd.Flags().StringVarP(&o.ExternalJenkinsBaseURL, "external-jenkins-url", "", "", "The jenkins url that an external git provider needs to use")
	cmd.Flags().BoolVarP(&o.DisableMaven, "disable-updatebot", "", false, "disable updatebot-maven-plugin from attempting to fix/update the maven pom.xml")
	cmd.Flags().StringVarP(&o.ImportMode, "import-mode", "m", "", fmt.Sprintf("The import mode to use. Should be one of %s", strings.Join(v1.ImportModeStrings, ", ")))
	cmd.Flags().BoolVarP(&o.UseDefaultGit, "use-default-git", "", false, "use default git account")
	cmd.Flags().StringVarP(&o.DeployKind, "deploy-kind", "", "", fmt.Sprintf("The kind of deployment to use for the project. Should be one of %s", strings.Join(deployKinds, ", ")))
	cmd.Flags().BoolVarP(&o.DeployOptions.Canary, opts.OptionCanary, "", false, "should we use canary rollouts (progressive delivery) by default for this application. e.g. using a Canary deployment via flagger. Requires the installation of flagger and istio/gloo in your cluster")
	cmd.Flags().BoolVarP(&o.DeployOptions.HPA, opts.OptionHPA, "", false, "should we enable the Horizontal Pod Autoscaler for this application.")
	cmd.Flags().BoolVarP(&o.Destination.JenkinsX.Enabled, "jx", "", false, "if you want to default to importing this project into Jenkins X instead of a Jenkins server if you have a mixed Jenkins X and Jenkins cluster")
	cmd.Flags().StringVarP(&o.Destination.JenkinsfileRunner.Image, "jenkinsfilerunner", "", "", "if you want to import into Jenkins X with Jenkinsfilerunner this argument lets you specify the container image to use")
	cmd.Flags().StringVar(&o.ServiceAccount, "service-account", "tekton-bot", "The Kubernetes ServiceAccount to use to run the initial pipeline")
	cmd.Flags().BoolVarP(&o.BatchMode, "batch-mode", "b", false, "Runs in batch mode without prompting for user input")

	opts.AddGitRepoOptionsArguments(cmd, &o.GitRepositoryOptions)
}

// Run executes the command
func (o *ImportOptions) Run() error {
	var err error
	jxClient, ns, err := o.JXClientAndDevNamespace()
	if err != nil {
		return err
	}

	err = o.DefaultsFromTeamSettings()
	if err != nil {
		return err
	}

	var userAuth *auth.UserAuth
	if o.GitProvider == nil {
		authConfigSvc, err := o.GitLocalAuthConfigService()
		if err != nil {
			return err
		}
		config := authConfigSvc.Config()
		var server *auth.AuthServer
		if o.RepoURL != "" {
			gitInfo, err := gits.ParseGitURL(o.RepoURL)
			if err != nil {
				return err
			}
			serverURL := gitInfo.HostURLWithoutUser()
			server = config.GetOrCreateServer(serverURL)
		} else {
			server, err = config.PickOrCreateServer(gits.GitHubURL, o.GitRepositoryOptions.ServerURL, "Which Git service do you wish to use", o.BatchMode, o.GetIOFileHandles())
			if err != nil {
				return err
			}
		}
		switch o.UseDefaultGit {
		case true:
			userAuth = config.CurrentUser(server, o.CommonOptions.InCluster())
		case false:
			switch o.GitRepositoryOptions.Username != "" {
			case true:
				userAuth = config.GetOrCreateUserAuth(server.URL, o.GitRepositoryOptions.Username)
				o.GetReporter().UsingGitUserName(o.GitRepositoryOptions.Username)
			case false:
				// Get the org in case there is more than one user auth on the server and batchMode is true
				org := o.getOrganisationOrCurrentUser()
				userAuth, err = config.PickServerUserAuth(server, "Git user name:", o.BatchMode, org, o.GetIOFileHandles())
				if err != nil {
					return err
				}
			}
		}
		if server.Kind == "" {
			server.Kind, err = o.GitServerHostURLKind(server.URL)
			if err != nil {
				return err
			}
		}
		if userAuth.IsInvalid() {
			f := func(username string) error {
				o.Git().PrintCreateRepositoryGenerateAccessToken(server, username, o.Out)
				return nil
			}
			if o.GitRepositoryOptions.ApiToken != "" {
				userAuth.ApiToken = o.GitRepositoryOptions.ApiToken
			}
			err = config.EditUserAuth(server.Label(), userAuth, userAuth.Username, false, o.BatchMode, f, o.GetIOFileHandles())
			if err != nil {
				return err
			}

			// TODO lets verify the auth works?
			if userAuth.IsInvalid() {
				return fmt.Errorf("Authentication has failed for user %v. Please check the user's access credentials and try again", userAuth.Username)
			}
		}
		err = authConfigSvc.SaveUserAuth(server.URL, userAuth)
		if err != nil {
			return fmt.Errorf("Failed to store git auth configuration %s", err)
		}

		o.GitServer = server
		o.GitUserAuth = userAuth
		o.GitProvider, err = gits.CreateProvider(server, userAuth, o.Git())
		if err != nil {
			return err
		}
	}

	if o.GitHub {
		return o.ImportProjectsFromGitHub()
	}

	if o.Dir == "" {
		args := o.Args
		if len(args) > 0 {
			o.Dir = args[0]
		} else {
			dir, err := os.Getwd()
			if err != nil {
				return err
			}
			o.Dir = dir
		}
	}

	checkForJenkinsfile := o.Jenkinsfile == ""
	shouldClone := checkForJenkinsfile || !o.DisableBuildPack

	if o.RepoURL != "" {
		if shouldClone {
			// Use the git user auth to clone the repo (needed for private repos etc)
			if o.GitUserAuth == nil {
				userAuth := o.GitProvider.UserAuth()
				o.GitUserAuth = &userAuth
			}
			o.RepoURL, err = o.Git().CreateAuthenticatedURL(o.RepoURL, o.GitUserAuth)
			if err != nil {
				return err
			}
			err = o.CloneRepository()
			if err != nil {
				return err
			}
		}
	} else {
		err = o.DiscoverGit()
		if err != nil {
			return err
		}

		if o.RepoURL == "" {
			err = o.DiscoverRemoteGitURL()
			if err != nil {
				return err
			}
		}
	}

	if o.AppName == "" {
		if o.RepoURL != "" {
			info, err := gits.ParseGitURL(o.RepoURL)
			if err != nil {
				log.Logger().Warnf("Failed to parse git URL %s : %s", o.RepoURL, err)
			} else {
				o.Organisation = info.Organisation
				o.AppName = info.Name
			}
		}
	}
	if o.AppName == "" {
		dir, err := filepath.Abs(o.Dir)
		if err != nil {
			return err
		}
		_, o.AppName = filepath.Split(dir)
	}
	o.AppName = naming.ToValidName(strings.ToLower(o.AppName))

	o.jenkinsClientFactory, err = factory.NewClientFactoryFromFactory(o.GetJXFactory())
	if err != nil {
		return errors.Wrapf(err, "failed to create the Jenkins ClientFactory")
	}

	jenkinsfile, err := o.HasJenkinsfile()
	if err != nil {
		return err
	}

	// lets pick the import destination
	o.Destination, err = o.PickImportDestination(o.jenkinsClientFactory, jenkinsfile)
	if err != nil {
		return err
	}

	if jenkinsfile != "" {
		if o.Destination.Jenkins.JenkinsName != "" || o.Destination.JenkinsfileRunner.Enabled {
			// lets not run the Jenkins X build packs
			o.DisableBuildPack = true
		}
	}

	if !o.DisableBuildPack {
		err = o.EvaluateBuildPack(jenkinsfile)
		if err != nil {
			return err
		}
	}
	err = o.fixDockerIgnoreFile()
	if err != nil {
		return err
	}

	err = o.fixMaven()
	if err != nil {
		return err
	}

	if o.RepoURL == "" {
		if !o.DryRun {
			err = o.CreateNewRemoteRepository()
			if err != nil {
				if !o.DisableBuildPack {
					log.Logger().Warn("Remote repository creation failed. In order to retry consider adding '--no-pack' option.")
				}
				return err
			}
		}
	} else {
		if shouldClone {
			err = o.Git().Push(o.Dir, "origin", false, "HEAD")
			if err != nil {
				return err
			}
		}
	}

	if o.DryRun {
		log.Logger().Info("dry-run so skipping import to Jenkins X")
		return nil
	}

	gitURL := ""
	if o.RepoURL != "" {
		gitInfo, err := gits.ParseGitURL(o.RepoURL)
		if err != nil {
			return err
		}
		gitURL = gitInfo.URLWithoutUser()
	}
	if gitURL == "" {
		// TODO do we really need this code now?
		gitURL = gits.SourceRepositoryProviderURL(o.GitProvider)
	}
	_, err = kube.GetOrCreateSourceRepository(jxClient, ns, o.AppName, o.Organisation, gitURL)
	if err != nil {
		return errors.Wrapf(err, "creating application resource for %s", util.ColorInfo(o.AppName))
	}

	if !o.GithubAppInstalled {
		githubAppMode, err := o.IsGitHubAppMode()
		if err != nil {
			return err
		}

		if githubAppMode {
			githubApp := &github.GithubApp{
				Factory: o.GetFactory(),
			}

			installed, err := githubApp.Install(o.Organisation, o.Repository, o.GetIOFileHandles(), false)
			if err != nil {
				return err
			}
			o.GithubAppInstalled = installed
		}
	}

	return o.doImport()
}

// GetJXFactory lazily creates a new jx factory
func (o *ImportOptions) GetJXFactory() jxfactory.Factory {
	if o.JXFactory == nil {
		o.JXFactory = jxfactory.NewFactory()
	}
	return o.JXFactory
}

// ImportProjectsFromGitHub import projects from github
func (o *ImportOptions) ImportProjectsFromGitHub() error {
	repos, err := gits.PickRepositories(o.GitProvider, o.Organisation, "Which repositories do you want to import", o.SelectAll, o.SelectFilter, o.GetIOFileHandles())
	if err != nil {
		return err
	}

	log.Logger().Info("Selected repositories")
	for _, r := range repos {
		o2 := ImportOptions{
			CommonOptions:    o.CommonOptions,
			Dir:              o.Dir,
			RepoURL:          r.CloneURL,
			Organisation:     o.Organisation,
			Repository:       r.Name,
			Jenkins:          o.Jenkins,
			GitProvider:      o.GitProvider,
			DisableBuildPack: o.DisableBuildPack,
		}
		log.Logger().Infof("Importing repository %s", util.ColorInfo(r.Name))
		err = o2.Run()
		if err != nil {
			return err
		}
	}
	return nil
}

// GetReporter returns the reporter interface
func (o *ImportOptions) GetReporter() ImportReporter {
	if o.reporter == nil {
		o.reporter = &LogImportReporter{}
	}
	return o.reporter
}

// SetReporter overrides the reporter interface
func (o *ImportOptions) SetReporter(reporter ImportReporter) {
	o.reporter = reporter
}

// HasJenkinsfile returns  the ile name if there is a Jenkinsfile or empty string if there is not
func (o *ImportOptions) HasJenkinsfile() (string, error) {
	dir := o.Dir
	var err error

	jenkinsfile := jenkinsfile.Name
	if o.Jenkinsfile != "" {
		jenkinsfile = o.Jenkinsfile
	}
	if !filepath.IsAbs(jenkinsfile) {
		jenkinsfile = filepath.Join(dir, jenkinsfile)
	}
	exists, err := util.FileExists(jenkinsfile)
	if err != nil {
		return "", err
	}
	if !exists {
		return "", nil
	}
	return jenkinsfile, nil
}

// EvaluateBuildPack performs an evaluation of the build pack on the current source
func (o *ImportOptions) EvaluateBuildPack(jenkinsfile string) error {
	// TODO this is a workaround of this draft issue:
	// https://github.com/Azure/draft/issues/476
	dir := o.Dir
	var err error

	args := &InvokeDraftPack{
		Dir:             dir,
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

	err = o.modifyDeployKind()
	if err != nil {
		return err
	}

	if o.PostDraftPackCallback != nil {
		err = o.PostDraftPackCallback()
		if err != nil {
			return err
		}
	}

	gitServerName, err := gits.GetHost(o.GitProvider)
	if err != nil {
		return err
	}

	if o.GitUserAuth == nil {
		userAuth := o.GitProvider.UserAuth()
		o.GitUserAuth = &userAuth
	}

	o.Organisation = o.GetOrganisation()
	if o.Organisation == "" {
		gitUsername := o.GitUserAuth.Username
		o.Organisation, err = gits.GetOwner(o.BatchMode, o.GitProvider, gitUsername, o.GetIOFileHandles())
		if err != nil {
			return err
		}
	}

	if o.AppName == "" {
		dir := o.Dir
		_, defaultRepoName := filepath.Split(dir)

		o.AppName, err = gits.GetRepoName(o.BatchMode, false, o.GitProvider, defaultRepoName, o.Organisation, o.GetIOFileHandles())
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

	err = o.Git().Add(dir, "*")
	if err != nil {
		return err
	}
	err = o.Git().CommitIfChanges(dir, "Jenkins X build pack")
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
	//walk through every file in the given dir and update the placeholders
	var currentUser string
	if o.GitServer != nil {
		currentUser = o.GitServer.CurrentUser
		if currentUser == "" {
			if o.GitProvider != nil {
				currentUser = o.GitProvider.CurrentUsername()
			}
		}
	}
	if currentUser == "" {
		log.Logger().Warn("No username defined for the current Git server!")
		currentUser = o.GitRepositoryOptions.Username
	}
	return currentUser
}

// GetOrganisation gets the organisation from the RepoURL (if in the github format of github.com/org/repo). It will
// do this in preference to the Organisation field (if set). If the repo URL does not implicitly specify an organisation
// then the Organisation specified in the options is used.
func (o *ImportOptions) GetOrganisation() string {
	org := ""
	gitInfo, err := gits.ParseGitURL(o.RepoURL)
	if err == nil && gitInfo.Organisation != "" {
		org = gitInfo.Organisation
		if o.Organisation != "" && org != o.Organisation {
			log.Logger().Warnf("organisation %s detected from URL %s. '--org %s' will be ignored", org, o.RepoURL, o.Organisation)
		}
	} else {
		org = o.Organisation
	}
	return org
}

// CreateNewRemoteRepository creates a new remote repository
func (o *ImportOptions) CreateNewRemoteRepository() error {
	authConfigSvc, err := o.GitLocalAuthConfigService()
	if err != nil {
		return err
	}

	dir := o.Dir
	_, defaultRepoName := filepath.Split(dir)

	o.GitRepositoryOptions.Owner = o.GetOrganisation()
	details := &o.GitDetails
	if details.RepoName == "" {
		details, err = gits.PickNewGitRepository(o.BatchMode, authConfigSvc, defaultRepoName, &o.GitRepositoryOptions,
			o.GitServer, o.GitUserAuth, o.Git(), o.GetIOFileHandles())
		if err != nil {
			return err
		}
	}

	repo, err := details.CreateRepository()
	if err != nil {
		return err
	}
	o.GitProvider = details.GitProvider

	o.RepoURL = repo.CloneURL
	pushGitURL, err := o.Git().CreateAuthenticatedURL(repo.CloneURL, details.User)
	if err != nil {
		return err
	}
	err = o.Git().AddRemote(dir, "origin", pushGitURL)
	if err != nil {
		return err
	}
	err = o.Git().PushMaster(dir)
	if err != nil {
		return err
	}
	repoURL := repo.HTMLURL
	o.GetReporter().PushedGitRepository(repoURL)

	githubAppMode, err := o.IsGitHubAppMode()
	if err != nil {
		return err
	}

	if !githubAppMode {

		// If the user creating the repo is not the pipeline user, add the pipeline user as a contributor to the repo
		if o.PipelineUserName != o.GitUserAuth.Username && o.GitServer != nil && o.GitServer.URL == o.PipelineServer {
			// Make the invitation
			err := o.GitProvider.AddCollaborator(o.PipelineUserName, details.Organisation, details.RepoName)
			if err != nil {
				return err
			}

			// If repo is put in an organisation that the pipeline user is not part of an invitation needs to be accepted.
			// Create a new provider for the pipeline user
			authConfig := authConfigSvc.Config()
			if err != nil {
				return err
			}
			pipelineUserAuth := authConfig.FindUserAuth(o.GitServer.URL, o.PipelineUserName)
			if pipelineUserAuth == nil {
				log.Logger().Warnf("Pipeline Git user credentials not found. %s will need to accept the invitation to collaborate"+
					"on %s if %s is not part of %s.\n",
					o.PipelineUserName, details.RepoName, o.PipelineUserName, details.Organisation)
			} else {
				pipelineServerAuth := authConfig.GetServer(authConfig.CurrentServer)
				pipelineUserProvider, err := gits.CreateProvider(pipelineServerAuth, pipelineUserAuth, o.Git())
				if err != nil {
					return err
				}

				// Get all invitations for the pipeline user
				// Wrapped in retry to not immediately fail the quickstart creation if APIs are flaky.
				f := func() error {
					invites, _, err := pipelineUserProvider.ListInvitations()
					if err != nil {
						return err
					}
					for _, x := range invites {
						// Accept all invitations for the pipeline user
						_, err = pipelineUserProvider.AcceptInvitation(*x.ID)
						if err != nil {
							return err
						}
					}
					return nil
				}
				exponentialBackOff := backoff.NewExponentialBackOff()
				timeout := 20 * time.Second
				exponentialBackOff.MaxElapsedTime = timeout
				exponentialBackOff.Reset()
				err = backoff.Retry(f, exponentialBackOff)
				if err != nil {
					return err
				}
			}

		}
	}

	return nil
}

// CloneRepository clones a repository
func (o *ImportOptions) CloneRepository() error {
	url := o.RepoURL
	if url == "" {
		return fmt.Errorf("no Git repository URL defined")
	}
	gitInfo, err := gits.ParseGitURL(url)
	if err != nil {
		return fmt.Errorf("failed to parse Git URL %s due to: %s", url, err)
	}
	if gitInfo.Host == gits.GitHubHost && strings.HasPrefix(gitInfo.Scheme, "http") {
		if !strings.HasSuffix(url, ".git") {
			url += ".git"
		}
		o.RepoURL = url
	}
	cloneDir, err := util.CreateUniqueDirectory(o.Dir, gitInfo.Name, util.MaximumNewDirectoryAttempts)
	if err != nil {
		return errors.Wrapf(err, "failed to create unique directory for '%s'", o.Dir)
	}
	err = o.Git().Clone(url, cloneDir)
	if err != nil {
		return errors.Wrapf(err, "failed to clone in directory '%s'", cloneDir)
	}
	o.Dir = cloneDir
	return nil
}

// DiscoverGit checks if there is a git clone or prompts the user to import it
func (o *ImportOptions) DiscoverGit() error {
	surveyOpts := survey.WithStdio(o.In, o.Out, o.Err)
	if !o.DisableDotGitSearch {
		root, gitConf, err := o.Git().FindGitConfigDir(o.Dir)
		if err != nil {
			return err
		}
		if root != "" {
			if root != o.Dir {
				o.GetReporter().Trace("Importing from directory %s as we found a .git folder there", root)
			}
			o.Dir = root
			o.GitConfDir = gitConf
			return nil
		}
	}

	dir := o.Dir
	if dir == "" {
		return fmt.Errorf("no directory specified")
	}

	// lets prompt the user to initialise the Git repository
	if !o.BatchMode {
		o.GetReporter().Trace("The directory %s is not yet using git", util.ColorInfo(dir))
		flag := false
		prompt := &survey.Confirm{
			Message: "Would you like to initialise git now?",
			Default: true,
		}
		err := survey.AskOne(prompt, &flag, nil, surveyOpts)
		if err != nil {
			return err
		}
		if !flag {
			return fmt.Errorf("please initialise git yourself then try again")
		}
	}
	o.InitialisedGit = true
	err := o.Git().Init(dir)
	if err != nil {
		return err
	}
	o.GitConfDir = filepath.Join(dir, ".git", "config")
	err = o.DefaultGitIgnore()
	if err != nil {
		return err
	}
	err = o.Git().Add(dir, ".gitignore")
	if err != nil {
		log.Logger().Debug("failed to add .gitignore")
	}
	err = o.Git().Add(dir, "*")
	if err != nil {
		return err
	}

	err = o.Git().Status(dir)
	if err != nil {
		return err
	}

	message := o.ImportGitCommitMessage
	if message == "" {
		if o.BatchMode {
			message = "Initial import"
		} else {
			messagePrompt := &survey.Input{
				Message: "Commit message: ",
				Default: "Initial import",
			}
			err = survey.AskOne(messagePrompt, &message, nil, surveyOpts)
			if err != nil {
				return err
			}
		}
	}
	err = o.Git().CommitIfChanges(dir, message)
	if err != nil {
		return err
	}
	o.GetReporter().GitRepositoryCreated()
	return nil
}

// DefaultGitIgnore creates a default .gitignore
func (o *ImportOptions) DefaultGitIgnore() error {
	name := filepath.Join(o.Dir, ".gitignore")
	exists, err := util.FileExists(name)
	if err != nil {
		return err
	}
	if !exists {
		data := []byte(opts.DefaultGitIgnoreFile)
		err = ioutil.WriteFile(name, data, util.DefaultWritePermissions)
		if err != nil {
			return fmt.Errorf("failed to write %s due to %s", name, err)
		}
	}
	return nil
}

// DiscoverRemoteGitURL finds the git url by looking in the directory
// and looking for a .git/config file
func (o *ImportOptions) DiscoverRemoteGitURL() error {
	gitConf := o.GitConfDir
	if gitConf == "" {
		return fmt.Errorf("no GitConfDir defined")
	}
	cfg := gitcfg.NewConfig()
	data, err := ioutil.ReadFile(gitConf)
	if err != nil {
		return fmt.Errorf("failed to load %s due to %s", gitConf, err)
	}

	err = cfg.Unmarshal(data)
	if err != nil {
		return fmt.Errorf("failed to unmarshal %s due to %s", gitConf, err)
	}
	remotes := cfg.Remotes
	if len(remotes) == 0 {
		return nil
	}
	url := o.Git().GetRemoteUrl(cfg, "origin")
	if url == "" {
		url = o.Git().GetRemoteUrl(cfg, "upstream")
		if url == "" {
			url, err = o.PickGitRemoteURL(cfg)
			if err != nil {
				return err
			}
		}
	}
	if url != "" {
		o.RepoURL = url
	}
	return nil
}

func (o *ImportOptions) doImport() error {
	gitURL := o.RepoURL
	gitProvider := o.GitProvider
	if gitProvider == nil {
		p, err := o.GitProviderForURL(gitURL, "user name to register webhook")
		if err != nil {
			return err
		}
		gitProvider = p
	}

	authConfigSvc, err := o.GitLocalAuthConfigService()
	if err != nil {
		return err
	}
	defaultJenkinsfileName := jenkinsfile.Name
	jenkinsfile := o.Jenkinsfile
	if jenkinsfile == "" {
		jenkinsfile = defaultJenkinsfileName
	}

	dockerfileLocation := ""
	if o.Dir != "" {
		dockerfileLocation = filepath.Join(o.Dir, "Dockerfile")
	} else {
		dockerfileLocation = "Dockerfile"
	}
	dockerfileExists, err := util.FileExists(dockerfileLocation)
	if err != nil {
		return err
	}

	if dockerfileExists {
		err = o.ensureDockerRepositoryExists()
		if err != nil {
			return err
		}
	}

	if !o.Destination.JenkinsX.Enabled {
		flag, err := util.Confirm("do you want to use ChatOps to trigger pipelines instead of jenkins webhooks?", false, "using ChatOps means lighthouse will handle webhooks and trigger jobs directly in Jenkins", o.GetIOFileHandles())
		if err != nil {
			return err
		}
		if flag {
			o.Destination.JenkinsX.Enabled = true

			err := o.enableTriggerPipelineJenkinsXPipeline(o.Destination)
			if err != nil {
				return err
			}
		}
	}

	if o.Destination.JenkinsX.Enabled {
		if o.Destination.JenkinsfileRunner.Enabled {
			err := o.enableJenkinsfileRunnerPipeline(o.Destination)
			if err != nil {
				return err
			}
		}

		log.Logger().Infof("importing the repository into Jenkins X")
		githubAppMode, err := o.IsGitHubAppMode()
		if err != nil {
			return err
		}

		if !o.DisableWebhooks && !githubAppMode {
			// register the webhook
			err = o.CreateWebhookProw(gitURL, gitProvider)
			if err != nil {
				return err
			}
		}
		return o.addProwConfig(gitURL, gitProvider.Kind())
	}

	// lets create the SourceRepository so we can trigger additional Jenkins X Pipelines against the Jenkins managed source repository
	if o.SchedulerName == "" {
		o.SchedulerName = "jenkins"
	}
	gitInfo, err := gits.ParseGitURL(gitURL)
	if err != nil {
		return err
	}
	_, err = o.getOrCreateSourceRepository(gitInfo, gitProvider.Kind())
	if err != nil {
		return err
	}

	log.Logger().Infof("importing the repository into Jenkins: %s", util.ColorInfo(o.Destination.Jenkins.JenkinsName))
	jc, err := o.jenkinsClientFactory.CreateJenkinsClient(o.Destination.Jenkins.JenkinsName)
	if err != nil {
		return errors.Wrapf(err, "failed to create jenkins client for %s", o.Destination.Jenkins.JenkinsName)
	}
	return o.ImportProjectIntoJenkins(jc, gitURL, o.Dir, jenkinsfile, o.BranchPattern, o.Credentials, false, gitProvider, authConfigSvc, false, o.BatchMode)
}

func (o *ImportOptions) addProwConfig(gitURL string, gitKind string) error {
	gitInfo, err := gits.ParseGitURL(gitURL)
	if err != nil {
		return err
	}
	repo := gitInfo.Organisation + "/" + gitInfo.Name
	client, err := o.KubeClient()
	if err != nil {
		return err
	}
	devEnv, settings, err := o.DevEnvAndTeamSettings()
	if err != nil {
		return err
	}
	_, currentNamespace, err := o.KubeClientAndNamespace()
	if err != nil {
		return err
	}

	gha, err := o.IsGitHubAppMode()
	if err != nil {
		return err
	}

	if settings.IsSchedulerMode() {
		sr, err := o.getOrCreateSourceRepository(gitInfo, gitKind)
		if err != nil {
			return err
		}

		sourceGitURL, err := kube.GetRepositoryGitURL(sr)
		if err != nil {
			return errors.Wrapf(err, "failed to get the git URL for SourceRepository %s", sr.Name)
		}

		devGitURL := devEnv.Spec.Source.URL
		if devGitURL != "" && !gha {
			// lets generate a PR
			base := devEnv.Spec.Source.Ref
			if base == "" {
				base = "master"
			}
			pro := &pr.StepCreatePrOptions{
				SrcGitURL:  sourceGitURL,
				GitURLs:    []string{devGitURL},
				Base:       base,
				Fork:       true,
				BranchName: sr.Name,
			}
			pro.CommonOptions = o.CommonOptions

			changeFn := func(dir string, gitInfo *gits.GitRepository) ([]string, error) {
				return nil, writeSourceRepoToYaml(dir, sr)
			}

			err := pro.CreatePullRequest("resource", changeFn)
			if err != nil {
				return errors.Wrapf(err, "failed to create Pull Request on the development environment git repository %s", devGitURL)
			}
			prURL := ""
			if pro.Results != nil && pro.Results.PullRequest != nil {
				prURL = pro.Results.PullRequest.URL
			}
			o.GetReporter().CreatedDevRepoPullRequest(prURL, devGitURL)
		}

		err = o.GenerateProwConfig(currentNamespace, devEnv)
		if err != nil {
			return err
		}
	} else {
		err = prow.AddApplication(client, []string{repo}, currentNamespace, o.Pack, settings)
		if err != nil {
			return err
		}
	}

	if !gha {
		startBuildOptions := start.StartPipelineOptions{
			CommonOptions: o.CommonOptions,
		}
		startBuildOptions.ServiceAccount = o.ServiceAccount
		startBuildOptions.Args = []string{fmt.Sprintf("%s/%s/%s", gitInfo.Organisation, gitInfo.Name, opts.MasterBranch)}
		err = startBuildOptions.Run()
		if err != nil {
			return fmt.Errorf("failed to start pipeline build: %s", err)
		}
	}

	o.LogImportedProject(false, gitInfo)

	return nil
}

func (o *ImportOptions) getOrCreateSourceRepository(gitInfo *gits.GitRepository, gitKind string) (*v1.SourceRepository, error) {
	jxClient, currentNamespace, err := o.JXClientAndDevNamespace()
	if err != nil {
		return nil, err
	}
	callback := func(sr *v1.SourceRepository) {
		u := gitInfo.CloneURL
		if strings.HasPrefix(gitInfo.URL, "http") {
			u = gitInfo.URLWithoutUser()
		}
		if u == "" {
			u = gitInfo.CloneURL
		}
		sr.Spec.ProviderKind = gitKind
		sr.Spec.Provider = gitInfo.HostURLWithoutUser()
		sr.Spec.URL = u
		if sr.Spec.URL == "" {
			sr.Spec.URL = gitInfo.HTMLURL
		}
		sr.Spec.HTTPCloneURL = u
		if sr.Spec.HTTPCloneURL == "" {
			sr.Spec.HTTPCloneURL = gitInfo.HTMLURL
		}
		sr.Spec.SSHCloneURL = gitInfo.SSHURL
	}
	sr, err := kube.GetOrCreateSourceRepositoryCallback(jxClient, currentNamespace, gitInfo.Name, gitInfo.Organisation, gitInfo.HostURLWithoutUser(), callback)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get or create the source repository callback")
	}
	log.Logger().Debugf("have SourceRepository: %s\n", sr.Name)

	// lets update the Scheduler if one is specified and its different to the default
	schedulerName := o.SchedulerName
	if schedulerName != "" && schedulerName != sr.Spec.Scheduler.Name {
		sr.Spec.Scheduler.Name = schedulerName
		_, err = jxClient.JenkinsV1().SourceRepositories(currentNamespace).Update(sr)
		if err != nil {
			log.Logger().Warnf("failed to update the SourceRepository %s to add the Scheduler name %s due to: %s\n", sr.Name, schedulerName, err.Error())
		}
	}
	return sr, nil
}

// writeSourceRepoToYaml marshals a SourceRepository to the given directory, making sure it can be loaded by boot.
func writeSourceRepoToYaml(dir string, sr *v1.SourceRepository) error {
	// lets check if we have a new jx 3 source repository
	outDir := filepath.Join(dir, "repositories", "templates")
	fileName := filepath.Join(outDir, sr.Name+"-sr.yaml")
	exists, err := util.DirExists(outDir)
	if err != nil {
		return errors.Wrapf(err, "failed to check if file exists %s", outDir)
	}
	if !exists {
		// lets default to the jx 3 location
		outDir = filepath.Join(dir, "src", "base", "namespaces", "jx", "source-repositories")
		fileName = filepath.Join(outDir, sr.Name+".yaml")
	}
	err = os.MkdirAll(outDir, util.DefaultWritePermissions)
	if err != nil {
		return errors.Wrapf(err, "failed to make directories %s", outDir)
	}

	// lets clear the fields we don't need to save
	clearSourceRepositoryMetadata(&sr.ObjectMeta)
	// Ensure it has the type information it needs
	sr.APIVersion = jenkinsio.GroupAndVersion
	sr.Kind = "SourceRepository"

	data, err := yaml.Marshal(&sr)
	if err != nil {
		return errors.Wrapf(err, "failed to marshal SourceRepository %s to yaml", sr.Name)
	}

	err = ioutil.WriteFile(fileName, data, util.DefaultWritePermissions)
	if err != nil {
		return errors.Wrapf(err, "failed to save SourceRepository file %s", fileName)
	}
	return nil
}

// clearSourceRepositoryMetadata clears unnecessary data
func clearSourceRepositoryMetadata(meta *metav1.ObjectMeta) {
	meta.CreationTimestamp.Time = time.Time{}
	meta.Namespace = ""
	meta.OwnerReferences = nil
	meta.Finalizers = nil
	meta.Generation = 0
	meta.GenerateName = ""
	meta.SelfLink = ""
	meta.UID = ""
	meta.ResourceVersion = ""

	for _, k := range removeSourceRepositoryAnnotations {
		delete(meta.Annotations, k)
	}
}

// ensureDockerRepositoryExists for some kinds of container registry we need to pre-initialise its use such as for ECR
func (o *ImportOptions) ensureDockerRepositoryExists() error {
	orgName := o.getOrganisationOrCurrentUser()
	appName := o.AppName
	if orgName == "" {
		log.Logger().Warnf("Missing organisation name!")
		return nil
	}
	if appName == "" {
		log.Logger().Warnf("Missing application name!")
		return nil
	}
	kubeClient, curNs, err := o.KubeClientAndNamespace()
	if err != nil {
		return err
	}
	ns, _, err := kube.GetDevNamespace(kubeClient, curNs)
	if err != nil {
		return err
	}

	region, _ := kube.ReadRegion(kubeClient, ns)
	cm, err := kubeClient.CoreV1().ConfigMaps(ns).Get(kube.ConfigMapJenkinsDockerRegistry, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("Could not find ConfigMap %s in namespace %s: %s", kube.ConfigMapJenkinsDockerRegistry, ns, err)
	}
	if cm.Data != nil {
		dockerRegistry := cm.Data["docker.registry"]
		if dockerRegistry != "" {
			if strings.HasSuffix(dockerRegistry, ".amazonaws.com") && strings.Index(dockerRegistry, ".ecr.") > 0 {
				return amazon.LazyCreateRegistry(kubeClient, ns, region, dockerRegistry, o.getDockerRegistryOrg(), appName)
			}
		}
	}
	return nil
}

// ReplacePlaceholders replaces app name, git server name, git org, and docker registry org placeholders
func (o *ImportOptions) ReplacePlaceholders(gitServerName, dockerRegistryOrg string) error {
	o.Organisation = naming.ToValidName(strings.ToLower(o.Organisation))
	o.GetReporter().Trace("replacing placeholders in directory %s", o.Dir)
	o.GetReporter().Trace("app name: %s, git server: %s, org: %s, Docker registry org: %s", o.AppName, gitServerName, o.Organisation, dockerRegistryOrg)

	ignore, err := gitignore.NewRepository(o.Dir)
	if err != nil {
		return err
	}

	replacer := strings.NewReplacer(
		util.PlaceHolderAppName, strings.ToLower(o.AppName),
		util.PlaceHolderGitProvider, strings.ToLower(gitServerName),
		util.PlaceHolderOrg, strings.ToLower(o.Organisation),
		util.PlaceHolderDockerRegistryOrg, strings.ToLower(dockerRegistryOrg))

	pathsToRename := []string{} // Renaming must be done post-Walk
	if err := filepath.Walk(o.Dir, func(f string, fi os.FileInfo, err error) error {
		if skip, err := o.skipPathForReplacement(f, fi, ignore); skip {
			return err
		}
		if strings.Contains(filepath.Base(f), util.PlaceHolderPrefix) {
			// Prepend so children are renamed before their parents
			pathsToRename = append([]string{f}, pathsToRename...)
		}
		if !fi.IsDir() {
			if err := replacePlaceholdersInFile(replacer, f); err != nil {
				return err
			}
		}
		return nil

	}); err != nil {
		return fmt.Errorf("error replacing placeholders %v", err)
	}

	for _, path := range pathsToRename {
		if err := replacePlaceholdersInPathBase(replacer, path); err != nil {
			return err
		}
	}
	return nil
}

func (o *ImportOptions) skipPathForReplacement(path string, fi os.FileInfo, ignore gitignore.GitIgnore) (bool, error) {
	relPath, _ := filepath.Rel(o.Dir, path)
	match := ignore.Relative(relPath, fi.IsDir())
	matchIgnore := match != nil && match.Ignore() //Defaults to including if match == nil
	if fi.IsDir() {
		if matchIgnore || fi.Name() == ".git" {
			o.GetReporter().Trace("skipping directory %q", path)
			return true, filepath.SkipDir
		}
	} else if matchIgnore {
		o.GetReporter().Trace("skipping ignored file %q", path)
		return true, nil
	}
	// Don't process nor follow symlinks
	if (fi.Mode() & os.ModeSymlink) == os.ModeSymlink {
		o.GetReporter().Trace("skipping symlink file %q", path)
		return true, nil
	}
	return false, nil
}

func replacePlaceholdersInFile(replacer *strings.Replacer, file string) error {
	input, err := ioutil.ReadFile(file)
	if err != nil {
		log.Logger().Errorf("failed to read file %s: %v", file, err)
		return err
	}

	lines := string(input)
	if strings.Contains(lines, util.PlaceHolderPrefix) { // Avoid unnecessarily rewriting files
		output := replacer.Replace(lines)
		err = ioutil.WriteFile(file, []byte(output), 0644)
		if err != nil {
			log.Logger().Errorf("failed to write file %s: %v", file, err)
			return err
		}
	}

	return nil
}

func replacePlaceholdersInPathBase(replacer *strings.Replacer, path string) error {
	base := filepath.Base(path)
	newBase := replacer.Replace(base)
	if newBase != base {
		newPath := filepath.Join(filepath.Dir(path), newBase)
		if err := os.Rename(path, newPath); err != nil {
			log.Logger().Errorf("failed to rename %q to %q: %v", path, newPath, err)
			return err
		}
	}
	return nil
}

func (o *ImportOptions) addAppNameToGeneratedFile(filename, field, value string) error {
	dir := filepath.Join(o.Dir, "charts", o.AppName)
	file := filepath.Join(dir, filename)
	exists, err := util.FileExists(file)
	if err != nil {
		return err
	}
	if !exists {
		// no file so lets ignore this
		return nil
	}
	input, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}

	lines := strings.Split(string(input), "\n")

	for i, line := range lines {
		if strings.Contains(line, field) {
			lines[i] = fmt.Sprintf("%s%s", field, value)
		}
	}
	output := strings.Join(lines, "\n")
	err = ioutil.WriteFile(file, []byte(output), 0644)
	if err != nil {
		return err
	}
	return nil
}

func (o *ImportOptions) renameChartToMatchAppName() error {
	var oldChartsDir string
	dir := o.Dir
	chartsDir := filepath.Join(dir, "charts")
	exists, err := util.DirExists(chartsDir)
	if err != nil {
		return errors.Wrapf(err, "failed to check if the charts directory exists %s", chartsDir)
	}
	if !exists {
		return nil
	}
	files, err := ioutil.ReadDir(chartsDir)
	if err != nil {
		return fmt.Errorf("error matching a Jenkins X build pack name with chart folder %v", err)
	}
	for _, fi := range files {
		if fi.IsDir() {
			name := fi.Name()
			// TODO we maybe need to try check if the sub dir named after the build pack matches first?
			if name != "preview" && name != ".git" {
				oldChartsDir = filepath.Join(chartsDir, name)
				break
			}
		}
	}
	if oldChartsDir != "" {
		// chart expects folder name to be the same as app name
		newChartsDir := filepath.Join(dir, "charts", o.AppName)

		exists, err := util.DirExists(oldChartsDir)
		if err != nil {
			return err
		}
		if exists && oldChartsDir != newChartsDir {
			err = util.RenameDir(oldChartsDir, newChartsDir, false)
			if err != nil {
				return fmt.Errorf("error renaming %s to %s, %v", oldChartsDir, newChartsDir, err)
			}
			_, err = os.Stat(newChartsDir)
			if err != nil {
				return err
			}
		}
		// now update the chart.yaml
		err = o.addAppNameToGeneratedFile("Chart.yaml", "name: ", o.AppName)
		if err != nil {
			return err
		}
	}
	return nil
}

func (o *ImportOptions) fixDockerIgnoreFile() error {
	filename := filepath.Join(o.Dir, ".dockerignore")
	exists, err := util.FileExists(filename)
	if err == nil && exists {
		data, err := ioutil.ReadFile(filename)
		if err != nil {
			return fmt.Errorf("Failed to load %s: %s", filename, err)
		}
		lines := strings.Split(string(data), "\n")
		for i, line := range lines {
			if strings.TrimSpace(line) == "Dockerfile" {
				lines = append(lines[:i], lines[i+1:]...)
				text := strings.Join(lines, "\n")
				err = ioutil.WriteFile(filename, []byte(text), util.DefaultWritePermissions)
				if err != nil {
					return err
				}
				o.GetReporter().Trace("Removed old `Dockerfile` entry from %s", util.ColorInfo(filename))
			}
		}
	}
	return nil
}

// CreateProwOwnersFile creates an OWNERS file in the root of the project assigning the current Git user as an approver and a reviewer. If the file already exists, does nothing.
func (o *ImportOptions) CreateProwOwnersFile() error {
	filename := filepath.Join(o.Dir, "OWNERS")
	exists, err := util.FileExists(filename)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	if o.GitUserAuth != nil && o.GitUserAuth.Username != "" {
		data := prow.Owners{
			Approvers: []string{o.GitUserAuth.Username},
			Reviewers: []string{o.GitUserAuth.Username},
		}
		yaml, err := yaml.Marshal(&data)
		if err != nil {
			return err
		}
		err = ioutil.WriteFile(filename, yaml, 0644)
		if err != nil {
			return err
		}
		return nil
	}
	return errors.New("GitUserAuth.Username not set")
}

// CreateProwOwnersAliasesFile creates an OWNERS_ALIASES file in the root of the project assigning the current Git user as an approver and a reviewer.
func (o *ImportOptions) CreateProwOwnersAliasesFile() error {
	filename := filepath.Join(o.Dir, "OWNERS_ALIASES")
	exists, err := util.FileExists(filename)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	if o.GitUserAuth == nil {
		return errors.New("option GitUserAuth not set")
	}
	gitUser := o.GitUserAuth.Username
	if gitUser != "" {
		data := prow.OwnersAliases{
			Aliases:       []string{gitUser},
			BestApprovers: []string{gitUser},
			BestReviewers: []string{gitUser},
		}
		yaml, err := yaml.Marshal(&data)
		if err != nil {
			return err
		}
		return ioutil.WriteFile(filename, yaml, 0644)
	}
	return errors.New("GitUserAuth.Username not set")
}

func (o *ImportOptions) fixMaven() error {
	if o.DisableMaven {
		return nil
	}
	dir := o.Dir
	pomName := filepath.Join(dir, "pom.xml")
	exists, err := util.FileExists(pomName)
	if err != nil {
		return err
	}
	if exists {
		err = maven.InstallMavenIfRequired()
		if err != nil {
			return err
		}

		// lets ensure the mvn plugins are ok
		out, err := o.GetCommandOutput(dir, "mvn", "io.jenkins.updatebot:updatebot-maven-plugin:RELEASE:plugin", "-Dartifact=maven-deploy-plugin", "-Dversion="+opts.MinimumMavenDeployVersion)
		if err != nil {
			return fmt.Errorf("Failed to update maven deploy plugin: %s output: %s", err, out)
		}
		out, err = o.GetCommandOutput(dir, "mvn", "io.jenkins.updatebot:updatebot-maven-plugin:RELEASE:plugin", "-Dartifact=maven-surefire-plugin", "-Dversion=3.0.0-M1")
		if err != nil {
			return fmt.Errorf("Failed to update maven surefire plugin: %s output: %s", err, out)
		}
		if !o.DryRun {
			err = o.Git().Add(dir, "pom.xml")
			if err != nil {
				return err
			}
			err = o.Git().CommitIfChanges(dir, "fix:(plugins) use a better version of maven plugins")
			if err != nil {
				return err
			}
		}

		// lets ensure the probe paths are ok
		out, err = o.GetCommandOutput(dir, "mvn", "io.jenkins.updatebot:updatebot-maven-plugin:RELEASE:chart")
		if err != nil {
			return fmt.Errorf("Failed to update chart: %s output: %s", err, out)
		}
		if !o.DryRun {
			exists, err := util.FileExists(filepath.Join(dir, "charts"))
			if err != nil {
				return err
			}
			if exists {
				err = o.Git().Add(dir, "charts")
				if err != nil {
					return err
				}
				err = o.Git().CommitIfChanges(dir, "fix:(chart) fix up the probe path")
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (o *ImportOptions) DefaultsFromTeamSettings() error {
	settings, err := o.TeamSettings()
	if err != nil {
		return err
	}
	return o.DefaultValuesFromTeamSettings(settings)
}

// DefaultValuesFromTeamSettings defaults the repository options from the given team settings
func (o *ImportOptions) DefaultValuesFromTeamSettings(settings *v1.TeamSettings) error {
	if o.DeployKind == "" {
		o.DeployKind = settings.DeployKind
	}

	// lets override any deploy o from the team settings if they are not specified
	teamDeployOptions := settings.GetDeployOptions()
	if !o.FlagChanged(opts.OptionCanary) {
		o.DeployOptions.Canary = teamDeployOptions.Canary
	}
	if !o.FlagChanged(opts.OptionHPA) {
		o.DeployOptions.HPA = teamDeployOptions.HPA
	}
	if o.Organisation == "" {
		o.Organisation = settings.Organisation
	}
	if o.GitRepositoryOptions.Owner == "" {
		o.GitRepositoryOptions.Owner = settings.Organisation
	}
	if o.DockerRegistryOrg == "" {
		o.DockerRegistryOrg = settings.DockerRegistryOrg
	}
	if o.GitRepositoryOptions.ServerURL == "" {
		o.GitRepositoryOptions.ServerURL = settings.GitServer
	}
	o.GitRepositoryOptions.Public = settings.GitPublic || o.GitRepositoryOptions.Public
	o.PipelineServer = settings.GitServer
	o.PipelineUserName = settings.PipelineUsername
	return nil
}

// ConfigureImportOptions updates the import options struct based on values from the create repo struct
func (o *ImportOptions) ConfigureImportOptions(repoData *gits.CreateRepoData) {
	// configure the import o based on previous answers
	o.AppName = repoData.RepoName
	o.GitProvider = repoData.GitProvider
	o.Organisation = repoData.Organisation
	o.Repository = repoData.RepoName
	o.GitDetails = *repoData
	o.GitServer = repoData.GitServer
}

// GetGitRepositoryDetails determines the git repository details to use during the import command
func (o *ImportOptions) GetGitRepositoryDetails() (*gits.CreateRepoData, error) {
	err := o.DefaultsFromTeamSettings()
	if err != nil {
		return nil, err
	}
	authConfigSvc, err := o.GitLocalAuthConfigService()
	if err != nil {
		return nil, err
	}
	//config git repositoryoptions parameters: Owner and RepoName
	o.GitRepositoryOptions.Owner = o.Organisation
	o.GitRepositoryOptions.RepoName = o.Repository
	details, err := gits.PickNewOrExistingGitRepository(o.BatchMode, authConfigSvc,
		"", &o.GitRepositoryOptions, nil, nil, o.Git(), false, o.GetIOFileHandles())
	if err != nil {
		return nil, err
	}
	return details, nil
}

// modifyDeployKind lets modify the deployment kind if the team settings or CLI settings are different
func (o *ImportOptions) modifyDeployKind() error {
	deployKind := o.DeployKind
	if deployKind == "" {
		return nil
	}
	dopts := o.DeployOptions

	copy := *o.CommonOptions
	cmd, eo := edit.NewCmdEditDeployKindAndOption(&copy)
	eo.Dir = o.Dir

	// lets parse the CLI arguments so that the flags are marked as specified to force them to be overridden
	err := cmd.Flags().Parse(edit.ToDeployArguments(opts.OptionKind, deployKind, dopts.Canary, dopts.HPA))
	if err != nil {
		return err
	}
	err = eo.Run()
	if err != nil {
		return errors.Wrapf(err, "failed to modify the deployment kind to %s", deployKind)
	}
	return nil
}

// enableTriggerPipelineJenkinsXPipeline lets generate the jenkins-x.yml if one doesn't exist
// lets use JENKINS_SERVER to point to the jenkins server to use
func (o *ImportOptions) enableTriggerPipelineJenkinsXPipeline(destination ImportDestination) error {
	projectConfig, fileName, err := config.LoadProjectConfig(o.Dir)
	if err != nil {
		return errors.Wrapf(err, "failed to load Jenkins X Pipeline in dir %s", o.Dir)
	}
	changed := false
	if projectConfig.BuildPack != triggerPipelineBuildPack {
		projectConfig.BuildPack = triggerPipelineBuildPack
		changed = true
	}
	if projectConfig.PipelineConfig == nil {
		projectConfig.PipelineConfig = &jenkinsfile.PipelineConfig{}
	}
	jenkinsServerName := destination.Jenkins.JenkinsName
	found := false
	for i, e := range projectConfig.PipelineConfig.Env {
		if e.Name == jenkinsServerEnvVar {
			if e.Value != jenkinsServerName {
				projectConfig.PipelineConfig.Env[i].Value = jenkinsServerName
				found = true
				changed = true
			}
		}
	}
	if !found {
		projectConfig.PipelineConfig.Env = append(projectConfig.PipelineConfig.Env, corev1.EnvVar{
			Name:  jenkinsServerEnvVar,
			Value: jenkinsServerName,
		})
		changed = true
	}
	if changed {
		err := projectConfig.SaveConfig(fileName)
		if err != nil {
			return err
		}
	}
	return nil
}

// enableJenkinsfileRunnerPipeline lets enable the JenkinfileRunner pipeline
func (o *ImportOptions) enableJenkinsfileRunnerPipeline(destination ImportDestination) error {
	projectConfig, fileName, err := config.LoadProjectConfig(o.Dir)
	if err != nil {
		return errors.Wrapf(err, "failed to load Jenkins X Pipeline in dir %s", o.Dir)
	}
	changed := false
	if projectConfig.BuildPack != jenkinsfileRunnerBuildPack {
		projectConfig.BuildPack = jenkinsfileRunnerBuildPack
		changed = true
	}
	imageName := destination.JenkinsfileRunner.Image
	if imageName != "" {
		// lets add override for the run steps image
		if projectConfig.PipelineConfig == nil {
			projectConfig.PipelineConfig = &jenkinsfile.PipelineConfig{}
		}

		stepType := syntax.StepOverrideReplace
		found := false
		for i, o := range projectConfig.PipelineConfig.Pipelines.Overrides {
			if o.Name == "run" {
				found = true
				step := o.Step
				if step == nil {
					step = &syntax.Step{}
				}
				if o.Step.Image != imageName {
					step.Image = imageName
					// not really necessary but is until https://github.com/jenkins-x/jx/issues/6739 is fixed
					step.Command = defaultJenkinsfileRunnerCommand

					projectConfig.PipelineConfig.Pipelines.Overrides[i].Step = step
					projectConfig.PipelineConfig.Pipelines.Overrides[i].Type = &stepType
					changed = true
				}
				break
			}
		}
		if !found {
			o := &syntax.PipelineOverride{
				Name: "run",
				Type: &stepType,
				Step: &syntax.Step{
					Image: imageName,

					// not really necessary but is until https://github.com/jenkins-x/jx/issues/6739 is fixed
					Command: defaultJenkinsfileRunnerCommand,
				},
			}
			projectConfig.PipelineConfig.Pipelines.Overrides = append(projectConfig.PipelineConfig.Pipelines.Overrides, o)
			changed = true
		}
	}
	if changed {
		err := projectConfig.SaveConfig(fileName)
		if err != nil {
			return err
		}
	}
	return nil
}

// PickBuildPackName if not in batch mode lets confirm to the user which build pack we are going to use
func (o *ImportOptions) PickBuildPackName(i *InvokeDraftPack, dir string, chosenPack string) (string, error) {
	if o.BatchMode || o.Pack != "" {
		return chosenPack, nil
	}
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return chosenPack, err
	}
	names := []string{}
	for _, f := range files {
		name := f.Name()
		if f.IsDir() && !strings.HasPrefix(name, ".") {
			names = append(names, name)
		}
	}

	name, err := util.PickNameWithDefault(names, "Confirm the build pack name you wish to use on this project", chosenPack,
		"the build pack name is used to determine the automated pipeline for your source code", o.GetIOFileHandles())
	return name, err
}
