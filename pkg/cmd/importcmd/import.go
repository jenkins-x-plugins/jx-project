package importcmd

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cenkalti/backoff"

	"github.com/denormal/go-gitignore"
	"github.com/jenkins-x-plugins/jx-project/pkg/cmd/common"
	"github.com/jenkins-x-plugins/jx-project/pkg/constants"
	"github.com/jenkins-x-plugins/jx-project/pkg/maven"
	"github.com/jenkins-x/go-scm/scm"
	v1 "github.com/jenkins-x/jx-api/v4/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx-api/v4/pkg/client/clientset/versioned"
	"github.com/jenkins-x/jx-helpers/v3/pkg/boot"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cmdrunner"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/templates"
	"github.com/jenkins-x/jx-helpers/v3/pkg/files"
	"github.com/jenkins-x/jx-helpers/v3/pkg/gitclient"
	"github.com/jenkins-x/jx-helpers/v3/pkg/gitclient/cli"
	"github.com/jenkins-x/jx-helpers/v3/pkg/gitclient/gitdiscovery"
	"github.com/jenkins-x/jx-helpers/v3/pkg/gitclient/giturl"
	"github.com/jenkins-x/jx-helpers/v3/pkg/input"
	"github.com/jenkins-x/jx-helpers/v3/pkg/input/inputfactory"
	"github.com/jenkins-x/jx-helpers/v3/pkg/kube"
	"github.com/jenkins-x/jx-helpers/v3/pkg/kube/jxclient"
	"github.com/jenkins-x/jx-helpers/v3/pkg/kube/jxenv"
	"github.com/jenkins-x/jx-helpers/v3/pkg/kube/naming"
	"github.com/jenkins-x/jx-helpers/v3/pkg/options"
	"github.com/jenkins-x/jx-helpers/v3/pkg/scmhelpers"
	"github.com/jenkins-x/jx-helpers/v3/pkg/termcolor"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/jenkins-x/lighthouse-client/pkg/repoowners"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/yaml"
)

// CallbackFn callback function
type CallbackFn func() error

// ImportOptions options struct for jx-project import
type ImportOptions struct {
	options.BaseOptions

	Args             []string
	RepoURL          string
	GitProviderURL   string
	DiscoveredGitURL string
	Dir              string
	Organisation     string
	Repository       string
	// Credentials                        string
	AppName      string
	SelectFilter string
	Jenkinsfile  string
	// BranchPattern                      string
	ImportGitCommitMessage string
	Pack                   string
	DockerRegistryOrg      string
	DeployKind             string
	SchedulerName          string
	GitConfDir             string
	PipelineUserName       string
	PipelineServer         string
	// ImportMode                         string
	ServiceAccount                     string
	Namespace                          string
	OperatorNamespace                  string
	BootSecretName                     string
	PipelineCatalogDir                 string
	DisableMaven                       bool
	GithubAppInstalled                 bool
	GitHub                             bool
	DryRun                             bool
	SelectAll                          bool
	DisableBuildPack                   bool
	DisableWebhooks                    bool
	DisableDotGitSearch                bool
	DisableStartPipeline               bool
	InitialisedGit                     bool
	WaitForSourceRepositoryPullRequest bool
	NoDevPullRequest                   bool
	IgnoreExistingRepository           bool
	IgnoreCollaborator                 bool
	PullRequestPollPeriod              time.Duration
	PullRequestPollTimeout             time.Duration
	DeployOptions                      v1.DeployOptions
	GitRepositoryOptions               scm.RepositoryInput
	KubeClient                         kubernetes.Interface
	JXClient                           versioned.Interface
	Input                              input.Interface
	ScmFactory                         scmhelpers.Factory
	Gitter                             gitclient.Interface
	CommandRunner                      cmdrunner.CommandRunner
	DevEnv                             *v1.Environment
	BootScmClient                      *scm.Client

	OnCompleteCallback    func() error
	PostDraftPackCallback CallbackFn
	gitInfo               *giturl.GitRepository
	Destination           ImportDestination
	reporter              ImportReporter
	PackFilter            func(*Pack)
	// env customization
	EnvName     string
	EnvStrategy string
	NestedRepo  bool

	/*
		TODO jenkins support
		Jenkins                            gojenkins.JenkinsClient
		jenkinsClientFactory *jenkinsutil.ClientFactory

	*/
}

const (
	updateBotMavenPluginVersion = "RELEASE"

	JenkinsfileName = "Jenkinsfile"
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

	deployKinds = []string{constants.DeployKindKnative, constants.DeployKindDefault}
)

// NewCmdImport the cobra command for jx-project import
func NewCmdImport() *cobra.Command {
	cmd, _ := NewCmdImportAndOptions()
	return cmd
}

// NewCmdImportAndOptions creates the cobra command for jx-project import and the options
func NewCmdImportAndOptions() (*cobra.Command, *ImportOptions) {
	opts := &ImportOptions{}

	cmd := &cobra.Command{
		Use:     "import",
		Short:   "Imports a local project or Git repository into Jenkins X",
		Long:    importLong,
		Example: fmt.Sprintf(importExample, common.BinaryName, common.BinaryName, common.BinaryName, common.BinaryName, common.BinaryName, common.BinaryName),
		Run: func(_ *cobra.Command, _ []string) {
			err := opts.Run()
			helper.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&opts.RepoURL, "url", "u", "", "The git clone URL to clone into the current directory and then import")
	cmd.Flags().BoolVarP(&opts.GitHub, "github", "", false, "If you wish to pick the repositories from GitHub to import")
	cmd.Flags().BoolVarP(&opts.SelectAll, "all", "", false, "If selecting projects to import from a Git provider this defaults to selecting them all")

	opts.AddImportFlags(cmd, false)
	return cmd, opts
}

func (o *ImportOptions) AddImportFlags(cmd *cobra.Command, createProject bool) {
	notCreateProject := func(text string) string {
		if createProject {
			return ""
		}
		return text
	}
	cmd.Flags().StringVarP(&o.GitProviderURL, "git-provider-url", "", "", "Deprecated: please use --git-server")
	cmd.Flags().StringVarP(&o.Organisation, "org", "", "", "Specify the Git provider organisation to import the project into (if it is not already in one)")
	cmd.Flags().StringVarP(&o.Dir, "dir", "", ".", "Specify the directory to import")
	cmd.Flags().StringVarP(&o.PipelineCatalogDir, "pipeline-catalog-dir", "", "", "The pipeline catalog directory you want to use instead of the buildPackGitURL in the dev Environment Team settings. Generally only used for testing pipelines")
	cmd.Flags().StringVarP(&o.Repository, "name", notCreateProject("n"), "", "Specify the Git repository name to import the project into (if it is not already in one)")
	cmd.Flags().BoolVarP(&o.DryRun, "dry-run", "", false, "Performs local changes to the repo but skips the import into Jenkins X")
	cmd.Flags().BoolVarP(&o.DisableBuildPack, "no-pack", "", false, "Disable trying to default a Dockerfile and Helm Chart from the pipeline catalog pack")
	cmd.Flags().BoolVarP(&o.DisableMaven, "no-maven-fix", "", false, "Disable trying to fix existing pom.xml")
	cmd.Flags().StringVarP(&o.ImportGitCommitMessage, "import-commit-message", "", "", "Specifies the initial commit message used when importing the project")
	cmd.Flags().StringVarP(&o.Pack, "pack", "", "", "The name of the pipeline catalog pack to use. If none is specified it will be chosen based on matching the source code languages")
	cmd.Flags().StringVarP(&o.DockerRegistryOrg, "docker-registry-org", "", "", "The name of the docker registry organisation to use. If not specified then the Git provider organisation will be used")
	cmd.Flags().StringVarP(&o.OperatorNamespace, "operator-namespace", "", boot.GitOperatorNamespace, "The namespace where the git operator is installed")
	cmd.Flags().StringVarP(&o.BootSecretName, "boot-secret-name", "", boot.SecretName, "The name of the boot secret")
	cmd.Flags().StringVarP(&o.DeployKind, "deploy-kind", "", "", fmt.Sprintf("The kind of deployment to use for the project. Should be one of %s", strings.Join(deployKinds, ", ")))
	cmd.Flags().BoolVarP(&o.DeployOptions.Canary, constants.OptionCanary, "", false, "should we use canary rollouts (progressive delivery) by default for this application. e.g. using a Canary deployment via flagger. Requires the installation of flagger and istio/gloo in your cluster")
	cmd.Flags().BoolVarP(&o.DeployOptions.HPA, constants.OptionHPA, "", false, "should we enable the Horizontal Pod Autoscaler for this application.")
	cmd.Flags().BoolVarP(&o.Destination.JenkinsX.Enabled, "jx", "", false, "if you want to default to importing this project into Jenkins X instead of a Jenkins server if you have a mixed Jenkins X and Jenkins cluster")
	cmd.Flags().StringVarP(&o.Destination.JenkinsfileRunner.Image, "jenkinsfilerunner", "", "", "if you want to import into Jenkins X with Jenkinsfilerunner this argument lets you specify the container image to use")
	cmd.Flags().StringVar(&o.ServiceAccount, "service-account", "tekton-bot", "The Kubernetes ServiceAccount to use to run the initial pipeline")
	cmd.Flags().StringVar(&o.SchedulerName, "scheduler", "in-repo", "Change schedulerName, More info about Scheduler: https://jenkins-x.io/v3/develop/faq/config/repos/#how-do-i-customise-a-scheduler")

	cmd.Flags().BoolVarP(&o.WaitForSourceRepositoryPullRequest, "wait-for-pr", "", true, "waits for the Pull Request generated on the cluster environment git repository to merge")
	cmd.Flags().BoolVarP(&o.NoDevPullRequest, "no-dev-pr", "", false, "disables generating a Pull Request on the cluster git repository")
	cmd.Flags().BoolVarP(&o.DisableStartPipeline, "no-start", "", false, "disables starting a release pipeline when importing/creating a new project")
	cmd.Flags().BoolVarP(&o.IgnoreCollaborator, "no-collaborator", "", false, "disables checking if the bot user is a collaborator. Only used if you have an issue with your git provider and this functionality in go-scm")
	cmd.Flags().DurationVarP(&o.PullRequestPollPeriod, "pr-poll-period", "", time.Second*20, "the time between polls of the Pull Request on the cluster environment git repository")
	cmd.Flags().DurationVarP(&o.PullRequestPollTimeout, "pr-poll-timeout", "", time.Minute*20, "the maximum amount of time we wait for the Pull Request on the cluster environment git repository")

	cmd.Flags().StringVar(&o.EnvName, "env-name", "", "The name of the environment to create (only used for env projects)")
	// FIXME parse enum and through what specified do not fit in enum
	cmd.Flags().StringVar(&o.EnvStrategy, "env-strategy", "Never", "The promotion strategy of the environment to create (only used for env projects)")
	cmd.Flags().BoolVarP(&o.NestedRepo, "nested-repo", "", false, "Specify if using nested repositories (in gitlab)")
	o.BaseOptions.AddBaseFlags(cmd)
	o.ScmFactory.AddFlags(cmd)

	cmd.Flags().StringVarP(&o.Destination.Jenkins.Server, "jenkins", "", "", "The name of the Jenkins server to import the project into")
}

// Validate validates the command line options
func (o *ImportOptions) Validate() error {
	err := o.BaseOptions.Validate()
	if err != nil {
		return errors.Wrapf(err, "failed to validate base options")
	}
	if o.Input == nil {
		o.Input = inputfactory.NewInput(&o.BaseOptions)
	}

	o.KubeClient, o.Namespace, err = kube.LazyCreateKubeClientAndNamespace(o.KubeClient, o.Namespace)
	if err != nil {
		return errors.Wrapf(err, "failed to create the kube client")
	}
	o.JXClient, err = jxclient.LazyCreateJXClient(o.JXClient)
	if err != nil {
		return errors.Wrapf(err, "failed to create the jx client")
	}
	if o.CommandRunner == nil {
		o.CommandRunner = cmdrunner.QuietCommandRunner
	}

	if o.DevEnv == nil {
		o.DevEnv, err = jxenv.GetDevEnvironment(o.JXClient, o.Namespace)
		if err != nil {
			return errors.Wrapf(err, "failed to find the dev Environment")
		}
	}
	if o.DevEnv == nil {
		extraMessage := ""
		if o.Namespace != "jx" {
			extraMessage = " Please run 'jx ns jx' to switch to the development namespace and retry this command"
		}
		return errors.Errorf("could not find the dev Environment in the namespace %s.%s", o.Namespace, extraMessage)
	}

	if o.ScmFactory.GitServerURL == "" && o.GitProviderURL != "" {
		o.ScmFactory.GitServerURL = o.GitProviderURL
	}
	if o.ScmFactory.GitServerURL == "" && o.gitInfo != nil {
		o.ScmFactory.GitServerURL = o.gitInfo.HostURL()
	}

	if o.ScmFactory.GitServerURL == "" {
		o.ScmFactory.GitServerURL, err = o.defaultGitServerURLFromDevEnv()
		if err != nil {
			return errors.Wrapf(err, "failed to default the git server URL from the dev Environment")
		}
	}
	if o.ScmFactory.GitServerURL == "" {
		return options.MissingOption("git-server")
	}

	if o.ScmFactory.GitKind == "" {
		o.ScmFactory.GitKind = giturl.SaasGitKind(o.ScmFactory.GitServerURL)
		if o.ScmFactory.GitKind == "" {
			log.Logger().Infof("no --git-kind supplied for server %s so assuming kind is github", o.ScmFactory.GitServerURL)
			o.ScmFactory.GitKind = "github"
		}
	}

	if o.ScmFactory.ScmClient == nil {
		if !o.BatchMode && o.ScmFactory.Input == nil {
			o.ScmFactory.Input = o.Input
		}
		o.ScmFactory.ScmClient, err = o.ScmFactory.Create()
		if err != nil {
			return errors.Wrapf(err, "failed to create ScmClient")
		}
	}

	if o.Dir == "" {
		dir, err := os.Getwd()
		if err != nil {
			return err
		}
		o.Dir = dir
	}
	return nil
}

// Run executes the command
func (o *ImportOptions) Run() error {
	err := o.Validate()
	if err != nil {
		return errors.Wrapf(err, "failed to validate options")
	}

	o.DiscoveredGitURL = o.RepoURL
	if o.RepoURL == "" {
		err = o.DiscoverGit()
		if err != nil {
			return err
		}

		o.DiscoveredGitURL, err = gitdiscovery.FindGitURLFromDir(o.Dir, true)
		if err != nil {
			return errors.Wrapf(err, "failed to discover the git URL")
		}
	}
	if o.DiscoveredGitURL != "" {
		o.gitInfo, err = giturl.ParseGitURL(o.DiscoveredGitURL)
		if err != nil {
			return err
		}
	}

	err = o.DefaultsFromTeamSettings()
	if err != nil {
		return err
	}

	/* TODO
	if o.GitHub {
		return o.ImportProjectsFromGitHub()
	}
	*/

	checkForJenkinsfile := o.Jenkinsfile == ""
	shouldClone := checkForJenkinsfile || !o.DisableBuildPack

	if o.RepoURL != "" {
		if shouldClone {
			o.RepoURL, err = o.ScmFactory.CreateAuthenticatedURL(o.RepoURL)
			if err != nil {
				return err
			}
			err = o.CloneRepository()
			if err != nil {
				return err
			}
		}
	}
	if o.AppName == "" && o.gitInfo != nil {
		o.Organisation = o.gitInfo.Organisation
		o.AppName = o.gitInfo.Name
	}
	if o.AppName == "" {
		dir, err := filepath.Abs(o.Dir)
		if err != nil {
			return err
		}
		_, o.AppName = filepath.Split(dir)
	}
	if o.Repository == "" && o.NestedRepo {
		o.Repository = o.AppName
	}
	o.AppName = naming.ToValidName(strings.ToLower(o.AppName))
	jenkinsfile, err := o.HasJenkinsfile()
	if err != nil {
		return err
	}

	devEnvCloneDir, err := o.CloneDevEnvironment()
	if err != nil {
		return errors.Wrapf(err, "failed to clone dev env git repository")
	}

	if jenkinsfile != "" {
		// let's pick the import destination for the jenkinsfile
		o.Destination, err = o.PickImportDestination(devEnvCloneDir)
		if err != nil {
			return err
		}
		if o.Destination.Jenkins.Server != "" {
			// let's not run the Jenkins X build packs
			o.DisableBuildPack = true
		} else if o.Destination.JenkinsfileRunner.Enabled {
			o.DisableBuildPack = false
			o.Pack = "jenkinsfilerunner"
		}
	}

	if !o.DisableBuildPack {
		g := filepath.Join(o.Dir, ".lighthouse", "*", "triggers.yaml")
		matches, err := filepath.Glob(g)
		if err != nil {
			return errors.Wrapf(err, "failed to evaluate glob %s", g)
		}
		if len(matches) > 0 {
			o.DisableBuildPack = true
		}
	}

	if !o.DisableBuildPack {
		err = o.EvaluateBuildPack(devEnvCloneDir, jenkinsfile)
		if err != nil {
			return err
		}
	}

	o.OnCompleteCallback = func() error {
		if !o.DisableBuildPack {
			log.Logger().Infof("committing the pipeline catalog changes...")
			_, err = gitclient.AddAndCommitFiles(o.Git(), o.Dir, "chore: Jenkins X build pack")
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
		if shouldClone {
			err = gitclient.Push(o.Git(), o.Dir, "origin", false, "HEAD")
			if err != nil {
				return err
			}
		}
		return nil
	}

	newRepository := false
	if o.DiscoveredGitURL == "" {
		if !o.DryRun {
			err = o.CreateNewRemoteRepository()
			if err != nil {
				if !o.DisableBuildPack {
					log.Logger().Warn("Remote repository creation failed. In order to retry consider adding '--no-pack' option.")
				}
				return err
			}
			newRepository = true
		}
	}
	if o.DryRun {
		shouldClone = false
		err = o.OnCompleteCallback()
		if err != nil {
			return errors.Wrapf(err, "failed to fix dockerfile and maven")
		}

		log.Logger().Info("dry-run so skipping actual import to Jenkins X")
		return nil
	}

	if !o.IgnoreCollaborator {
		err = o.AddAndAcceptCollaborator(newRepository)
		if err != nil {
			return errors.Wrapf(err, "failed to add and accept collaborator")
		}
	}

	gitURL := ""
	if o.DiscoveredGitURL != "" {
		gitInfo, err := giturl.ParseGitURL(o.DiscoveredGitURL)
		if err != nil {
			return err
		}
		gitURL = gitInfo.URLWithoutUser()
	}
	if gitURL == "" {
		return errors.Errorf("no git URL could be found")
	}

	/* TODO github app support
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
	*/
	return o.doImport()
}

// ImportProjectsFromGitHub import projects from github
/** TODO
func (o *ImportOptions) ImportProjectsFromGitHub() error {
	repos, err := gits.PickRepositories(o.ScmClient, o.Organisation, "Which repositories do you want to import", o.SelectAll, o.SelectFilter, o.GetIOFileHandles())
	if err != nil {
		return err
	}

	log.Logger().Info("Selected repositories")
	for _, r := range repos {
		o2 := ImportOptions{
			CommonOptions: o.CommonOptions,
			Dir:           o.Dir,
			RepoURL:       r.CloneURL,
			Organisation:  o.Organisation,
			Repository:    r.Name,
			//Jenkins:          o.Jenkins,
			ScmClient:        o.ScmClient,
			DisableBuildPack: o.DisableBuildPack,
		}
		log.Logger().Infof("Importing repository %s", termcolor.ColorInfo(r.Name))
		err = o2.Run()
		if err != nil {
			return err
		}
	}
	return nil
}
*/

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

// CreateNewRemoteRepository creates a new remote repository
func (o *ImportOptions) CreateNewRemoteRepository() error {
	dir := o.Dir
	_, defaultRepoName := filepath.Split(dir)

	var err error
	o.GitRepositoryOptions.Namespace = o.GetOrganisation()
	details := &o.GitRepositoryOptions
	if o.Organisation == "" {
		o.Organisation, err = o.PickOwner("")
		if err != nil {
			return errors.Wrapf(err, "failed to pick owner")
		}

	}
	if details.Name == "" {
		details.Name, err = o.PickRepoName(o.Organisation, defaultRepoName, false)
		if err != nil {
			return errors.Wrapf(err, "failed to pick repository name")
		}
	}
	ctx := context.Background()
	createRepo := o.GitRepositoryOptions

	// need to clear the owner if its a user
	if o.getCurrentUser() == createRepo.Namespace {
		createRepo.Namespace = ""
	}
	repo, _, err := o.ScmFactory.ScmClient.Repositories.Create(ctx, &createRepo)
	if err != nil {
		return errors.Wrapf(err, "failed to create git repository %s/%s", o.GitRepositoryOptions.Namespace, o.GitRepositoryOptions.Name)
	}

	if err != nil {
		return err
	}

	// mostly to default a value in test cases if its missing
	if repo.Clone == "" {
		repo.Clone = repo.Link
	}

	// let's allow a BDD test to switch the git host to push to
	// e.g. if using kind and gitea and running tests inside k8s without public access to the gitea server
	gitPushHost := os.Getenv("JX_GIT_PUSH_HOST")
	if repo.Clone != "" && gitPushHost != "" {
		u, err := url.Parse(repo.Clone)
		if err != nil {
			return errors.Wrapf(err, "failed to parse repository clone URL %s", repo.Clone)
		}
		u.Host = gitPushHost
		repo.Clone = u.String()
		log.Logger().Infof("switching to the git clone URL %s", info(repo.Clone))
	}

	o.DiscoveredGitURL = repo.Clone
	pushGitURL, err := o.ScmFactory.CreateAuthenticatedURL(repo.Clone)
	if err != nil {
		return err
	}
	err = gitclient.AddRemote(o.Git(), dir, "origin", pushGitURL)
	if err != nil {
		return err
	}

	// let's use a retry loop to push in case the repository is not yet setup quite yet
	f := func() error {
		return gitclient.Push(o.Git(), dir, "origin", false, "HEAD")
	}

	bo := backoff.NewExponentialBackOff()
	bo.InitialInterval = 3 * time.Second
	bo.MaxElapsedTime = time.Minute
	bo.Reset()
	err = backoff.Retry(f, bo)
	if err != nil {
		return err
	}
	repoURL := repo.Link
	o.GetReporter().PushedGitRepository(repoURL)
	return nil
}

// CloneRepository clones a repository
func (o *ImportOptions) CloneRepository() error {
	repoURL := o.RepoURL
	if repoURL == "" {
		return fmt.Errorf("no Git repository URL defined")
	}
	gitInfo, err := giturl.ParseGitURL(repoURL)
	if err != nil {
		return fmt.Errorf("failed to parse Git URL %s due to: %s", repoURL, err)
	}
	if gitInfo.Host == giturl.GitHubHost && strings.HasPrefix(gitInfo.Scheme, "http") {
		if !strings.HasSuffix(repoURL, ".git") {
			repoURL += ".git"
		}
		o.RepoURL = repoURL
	}

	cloneDir, err := files.CreateUniqueDirectory(o.Dir, gitInfo.Name, files.MaximumNewDirectoryAttempts)
	if err != nil {
		return errors.Wrapf(err, "failed to create unique directory for '%s'", o.Dir)
	}
	cloneDir, err = gitclient.CloneToDir(o.Git(), repoURL, cloneDir)
	if err != nil {
		return errors.Wrapf(err, "failed to clone in directory '%s'", cloneDir)
	}
	o.Dir = cloneDir
	return nil
}

// DiscoverGit checks if there is a git clone or prompts the user to import it
func (o *ImportOptions) DiscoverGit() error {
	if !o.DisableDotGitSearch {
		root, gitConf, err := gitclient.FindGitConfigDir(o.Dir)
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

	// let's prompt the user to initialise the Git repository
	if !o.BatchMode {
		o.GetReporter().Trace("The directory %s is not yet using git", termcolor.ColorInfo(dir))

		flag, err := o.Input.Confirm("Would you like to initialise git now?", true, "We need to initialise git in the directory to continue")
		if err != nil {
			return errors.Wrapf(err, "failed to confirm git initialise")
		}
		if !flag {
			return fmt.Errorf("please initialise git yourself then try again")
		}
	}
	o.InitialisedGit = true
	err := gitclient.Init(o.Git(), dir)
	if err != nil {
		return err
	}
	o.GitConfDir = filepath.Join(dir, ".git", "config")
	err = o.DefaultGitIgnore()
	if err != nil {
		return err
	}
	err = gitclient.Add(o.Git(), dir, ".gitignore")
	if err != nil {
		log.Logger().Debug("failed to add .gitignore")
	}
	err = gitclient.Add(o.Git(), dir, "*")
	if err != nil {
		return err
	}

	_, err = gitclient.Status(o.Git(), dir)
	if err != nil {
		return err
	}

	message := o.ImportGitCommitMessage
	if message == "" {
		if o.BatchMode {
			message = "chore: initial import"
		} else {
			message, err = o.Input.PickValue("Commit message: ", "chore: initial import", true, "Please enter the initial git commit message")
			if err != nil {
				return errors.Wrapf(err, "failed to confirm commit message")
			}
		}
	}
	err = gitclient.CommitIfChanges(o.Git(), dir, message)
	if err != nil {
		return err
	}
	o.GetReporter().GitRepositoryCreated()
	return nil
}

// DefaultGitIgnore creates a default .gitignore
func (o *ImportOptions) DefaultGitIgnore() error {
	name := filepath.Join(o.Dir, ".gitignore")
	exists, err := files.FileExists(name)
	if err != nil {
		return err
	}
	if !exists {
		data := []byte(constants.DefaultGitIgnoreFile)
		err = os.WriteFile(name, data, files.DefaultFileWritePermissions)
		if err != nil {
			return fmt.Errorf("failed to write %s due to %s", name, err)
		}
	}
	return nil
}

func (o *ImportOptions) doImport() error {
	gitURL := o.DiscoveredGitURL

	// TODO should we prompt the user for the git kind if we can't detect / find it?
	gitKind := o.ScmFactory.GitKind

	remoteCluster, err := o.addSourceConfigPullRequest(gitURL, gitKind)
	if err != nil {
		return errors.Wrapf(err, "failed to create Pull Request on the cluster git repository")
	}

	if o.DisableStartPipeline {
		return nil
	}

	repoName := o.GitRepositoryOptions.Name
	if repoName == "" {
		repoName = o.AppName
	}
	repoFullName := scm.Join(o.Organisation, repoName)

	if !o.Destination.Jenkins.Enabled && !remoteCluster {
		c := &cmdrunner.Command{
			Name: "jx",
			Args: []string{"pipeline", "wait", "--owner", o.Organisation, "--repo", repoName},
			Out:  os.Stdout,
			Err:  os.Stderr,
		}
		_, err = o.CommandRunner(c)
		if err != nil {
			return errors.Wrapf(err, "failed to wait for the pipeline to be setup %s", repoFullName)
		}
	}

	// let's git push the build pack changes now to trigger a release
	//
	// TODO we could make this an optional Pull request etc?
	if o.OnCompleteCallback != nil {
		err = o.OnCompleteCallback()
		if err != nil {
			return errors.Wrapf(err, "failed to push git changes")
		}
	}

	if o.Destination.Jenkins.Enabled {
		return nil
	}

	log.Logger().Info("")
	log.Logger().Infof("Pipeline should start soon for: %s", info(repoFullName))
	log.Logger().Info("")
	log.Logger().Infof("Watch pipeline activity via:    %s", info(fmt.Sprintf("jx get activity -f %s -w", repoFullName)))
	log.Logger().Infof("Browse the pipeline log via:    %s", info(fmt.Sprintf("jx get build logs %s", repoFullName)))
	log.Logger().Infof("You can list the pipelines via: %s", info("jx get pipelines"))
	log.Logger().Infof("When the pipeline is complete:  %s", info("jx get applications"))
	log.Logger().Info("")
	log.Logger().Infof("For more help on available commands see: %s", info("https://jenkins-x.io/developing/browsing/"))
	log.Logger().Info("")

	return nil
}

// ReplacePlaceholders replaces app name, git server name, git org, and docker registry org placeholders
func (o *ImportOptions) ReplacePlaceholders(gitServerName, dockerRegistryOrg string) error {
	safeOrganisationName := naming.ToValidName(strings.ToLower(o.Organisation))
	o.GetReporter().Trace("replacing placeholders in directory %s", o.Dir)
	o.GetReporter().Trace("app name: %s, git server: %s, org: %s, Docker registry org: %s", o.AppName, gitServerName, o.Organisation, dockerRegistryOrg)

	ignore, err := gitignore.NewRepository(o.Dir)
	if err != nil {
		return err
	}

	replacer := strings.NewReplacer(
		constants.PlaceHolderAppName, strings.ToLower(o.AppName),
		constants.PlaceHolderGitProvider, strings.ToLower(gitServerName),
		constants.PlaceHolderOrg, safeOrganisationName,
		constants.PlaceHolderDockerRegistryOrg, strings.ToLower(dockerRegistryOrg))

	pathsToRename := []string{} // Renaming must be done post-Walk
	if err := filepath.Walk(o.Dir, func(f string, fi os.FileInfo, _ error) error {
		if skip, err := o.skipPathForReplacement(f, fi, ignore); skip {
			return err
		}
		if strings.Contains(filepath.Base(f), constants.PlaceHolderPrefix) {
			// Prepend so children are renamed before their parents
			pathsToRename = append([]string{f}, pathsToRename...)
		}
		if !fi.IsDir() {
			// TODO: Apply  https://docs.gomplate.ca/ if .jx/gotemplate.yaml exists
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
	matchIgnore := match != nil && match.Ignore() // Defaults to including if match == nil
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
	fileContent, err := os.ReadFile(file)
	if err != nil {
		log.Logger().Errorf("failed to read file %s: %v", file, err)
		return err
	}

	lines := string(fileContent)
	if strings.Contains(lines, constants.PlaceHolderPrefix) { // Avoid unnecessarily rewriting files
		output := replacer.Replace(lines)
		err = os.WriteFile(file, []byte(output), 0600)
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
	exists, err := files.FileExists(file)
	if err != nil {
		return err
	}
	if !exists {
		// no file so lets ignore this
		return nil
	}
	fileContent, err := os.ReadFile(file)
	if err != nil {
		return err
	}

	lines := strings.Split(string(fileContent), "\n")

	for i, line := range lines {
		if strings.HasPrefix(line, field) {
			lines[i] = fmt.Sprintf("%s%s", field, value)
		}
	}
	output := strings.Join(lines, "\n")
	err = os.WriteFile(file, []byte(output), 0600)
	if err != nil {
		return err
	}
	return nil
}

func (o *ImportOptions) renameChartToMatchAppName() error {
	var oldChartsDir string
	dir := o.Dir
	chartsDir := filepath.Join(dir, "charts")
	exists, err := files.DirExists(chartsDir)
	if err != nil {
		return errors.Wrapf(err, "failed to check if the charts directory exists %s", chartsDir)
	}
	if !exists {
		return nil
	}
	fileSlice, err := os.ReadDir(chartsDir)
	if err != nil {
		return fmt.Errorf("error matching a Jenkins X build pack name with chart folder %v", err)
	}
	for _, fi := range fileSlice {
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

		exists, err := files.DirExists(oldChartsDir)
		if err != nil {
			return err
		}
		if exists && oldChartsDir != newChartsDir {
			err = files.RenameDir(oldChartsDir, newChartsDir, false)
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
	exists, err := files.FileExists(filename)
	if err == nil && exists {
		data, err := os.ReadFile(filename)
		if err != nil {
			return fmt.Errorf("failed to load %s: %s", filename, err)
		}
		lines := strings.Split(string(data), "\n")
		for i, line := range lines {
			if strings.TrimSpace(line) != "Dockerfile" {
				continue
			}
			lines = append(lines[:i], lines[i+1:]...)
			text := strings.Join(lines, "\n")
			err = os.WriteFile(filename, []byte(text), files.DefaultFileWritePermissions)
			if err != nil {
				return err
			}
			o.GetReporter().Trace("Removed old `Dockerfile` entry from %s", termcolor.ColorInfo(filename))

		}
	}
	return nil
}

// CreateProwOwnersFile creates an OWNERS file in the root of the project assigning the current Git user as an approver and a reviewer. If the file already exists, does nothing.
func (o *ImportOptions) CreateProwOwnersFile() error {
	filename := filepath.Join(o.Dir, "OWNERS")
	exists, err := files.FileExists(filename)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	userName := o.getCurrentUser()
	if userName == "" {
		return errors.Errorf("no git username")
	}
	data := repoowners.SimpleConfig{
		Config: repoowners.Config{
			Approvers: []string{userName},
			Reviewers: []string{userName},
		},
	}
	yamlBytes, err := yaml.Marshal(&data)
	if err != nil {
		return err
	}
	err = os.WriteFile(filename, yamlBytes, 0600)
	if err != nil {
		return err
	}
	return nil
}

// CreateProwOwnersAliasesFile creates an OWNERS_ALIASES file in the root of the project assigning the current Git user as an approver and a reviewer.
func (o *ImportOptions) CreateProwOwnersAliasesFile() error {
	filename := filepath.Join(o.Dir, "OWNERS_ALIASES")
	exists, err := files.FileExists(filename)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	gitUser := o.getCurrentUser()
	if gitUser == "" {
		return errors.Errorf("no git username")
	}
	data := repoowners.OwnerAliases{
		Aliases: map[string][]string{
			"best-approvers": {gitUser},
			"best-reviewers": {gitUser},
		},
	}
	yamlBytes, err := yaml.Marshal(&data)
	if err != nil {
		return err
	}
	return os.WriteFile(filename, yamlBytes, 0600)
}

func (o *ImportOptions) fixMaven() error {
	if o.DisableMaven {
		return nil
	}
	dir := o.Dir
	pomName := filepath.Join(dir, "pom.xml")
	exists, err := files.FileExists(pomName)
	if err != nil {
		return err
	}
	if exists {
		err = maven.InstallMavenIfRequired(o.CommandRunner)
		if err != nil {
			return err
		}

		// let's ensure the mvn plugins are ok
		out, err := o.CommandRunner(cmdrunner.NewCommand(dir, "mvn", "io.jenkins.updatebot:updatebot-maven-plugin:"+updateBotMavenPluginVersion+":plugin", "-Dartifact=maven-deploy-plugin", "-Dversion="+constants.MinimumMavenDeployVersion))
		if err != nil {
			return fmt.Errorf("failed to update maven deploy plugin: %s output: %s", err, out)
		}
		out, err = o.CommandRunner(cmdrunner.NewCommand(dir, "mvn", "io.jenkins.updatebot:updatebot-maven-plugin:"+updateBotMavenPluginVersion+":plugin", "-Dartifact=maven-surefire-plugin", "-Dversion=3.0.0-M1"))
		if err != nil {
			return fmt.Errorf("failed to update maven surefire plugin: %s output: %s", err, out)
		}
		_, err = gitclient.AddAndCommitFiles(o.Git(), dir, "fix(plugins): use a better version of maven plugins")
		if err != nil {
			return err
		}

		// let's ensure the probe paths are ok
		out, err = o.CommandRunner(cmdrunner.NewCommand(dir, "mvn", "io.jenkins.updatebot:updatebot-maven-plugin:"+updateBotMavenPluginVersion+":chart"))
		if err != nil {
			return fmt.Errorf("failed to update chart: %s output: %s", err, out)
		}
		if out != "" {
			log.Logger().Info(out)
		}
		exists, err := files.FileExists(filepath.Join(dir, "charts"))
		if err != nil {
			return err
		}
		if exists {
			_, err = gitclient.AddAndCommitFiles(o.Git(), dir, "fix(chart): fix up the probe path")
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (o *ImportOptions) DefaultsFromTeamSettings() error {
	settings, err := jxenv.GetDevEnvTeamSettings(o.JXClient, o.Namespace)
	if err != nil {
		return errors.Wrapf(err, "failed to load Team Settings")
	}
	return o.DefaultValuesFromTeamSettings(settings)
}

// DefaultValuesFromTeamSettings defaults the repository options from the given team settings
func (o *ImportOptions) DefaultValuesFromTeamSettings(settings *v1.TeamSettings) error {
	if o.DeployKind == "" {
		o.DeployKind = settings.DeployKind
	}

	// let's override any deploy o from the team settings if they are not specified
	/* TODO
	teamDeployOptions := settings.GetDeployOptions()
	if !o.FlagChanged(OptionCanary) {
		o.DeployOptions.Canary = teamDeployOptions.Canary
	}
	if !o.FlagChanged(OptionHPA) {
		o.DeployOptions.HPA = teamDeployOptions.HPA
	}
	*/
	if o.Organisation == "" {
		o.Organisation = settings.Organisation
	}
	if o.GitRepositoryOptions.Namespace == "" {
		o.GitRepositoryOptions.Namespace = settings.Organisation
	}
	if o.DockerRegistryOrg == "" {
		o.DockerRegistryOrg = settings.DockerRegistryOrg
	}
	if o.ScmFactory.GitServerURL == "" {
		o.ScmFactory.GitServerURL = settings.GitServer
	}
	o.GitRepositoryOptions.Private = !settings.GitPublic
	o.PipelineServer = settings.GitServer
	o.PipelineUserName = settings.PipelineUsername
	return nil
}

// ConfigureImportOptions updates the import options struct based on values from the create repo struct
func (o *ImportOptions) ConfigureImportOptions(repoData *CreateRepoData) {
	// configure the import options based on previous answers
	owner := repoData.Organisation
	repoName := repoData.RepoName

	o.Organisation = owner
	o.AppName = repoName
	o.Repository = repoName
	o.GitRepositoryOptions.Namespace = owner
	o.GitRepositoryOptions.Name = repoName
}

// GetGitRepositoryDetails determines the git repository details to use during the import command
func (o *ImportOptions) GetGitRepositoryDetails() (*CreateRepoData, error) {
	err := o.DefaultsFromTeamSettings()
	if err != nil {
		return nil, err
	}
	// config git repositoryoptions parameters: Owner and RepoName
	o.GitRepositoryOptions.Namespace = o.Organisation
	o.GitRepositoryOptions.Name = o.Repository
	details, err := o.PickNewOrExistingGitRepository()
	if err != nil {
		return nil, err
	}
	return details, nil
}

// PickCatalogFolderName if not in batch mode lets confirm to the user which catalog folder we are going to use
func (o *ImportOptions) PickCatalogFolderName(dir, chosenPack string) (string, error) {
	if o.BatchMode || o.Pack != "" {
		return chosenPack, nil
	}
	fileList, err := os.ReadDir(dir)
	if err != nil {
		return chosenPack, err
	}
	names := []string{}
	for _, f := range fileList {
		name := f.Name()
		if f.IsDir() && !strings.HasPrefix(name, ".") {
			names = append(names, name)
		}
	}

	name, err := o.Input.PickNameWithDefault(names, "Confirm the catalog folder you wish to use on this project", chosenPack,
		"the catalog folder contains the tekton pipelines and associated files to be used on your source code")
	return name, err
}

// Git returns the gitter - lazily creating one if required
func (o *ImportOptions) Git() gitclient.Interface {
	if o.Gitter == nil {
		o.Gitter = cli.NewCLIClient("", o.CommandRunner)
	}
	return o.Gitter
}

func (o *ImportOptions) waitForSourceRepositoryPullRequest(pullRequestInfo *scm.PullRequest) error {
	start := time.Now()
	end := start.Add(o.PullRequestPollTimeout)
	durationString := o.PullRequestPollTimeout.String()

	if o.PullRequestPollPeriod == 0 {
		o.PullRequestPollPeriod = time.Second * 20
	}
	count := 0
	if pullRequestInfo != nil {
		log.Logger().Infof("Waiting up to %s for the pull request %s to merge with poll period %v....", durationString, termcolor.ColorInfo(pullRequestInfo.Link), o.PullRequestPollPeriod.String())
		count++
		defer log.Logger().Debugf("pull request poll count: %d", count)

		ctx := context.Background()
		fullName := pullRequestInfo.Repository().FullName
		prNumber := pullRequestInfo.Number
		for {
			pr, _, err := o.ScmFactory.ScmClient.PullRequests.Find(ctx, fullName, prNumber)
			if err != nil {
				log.Logger().Warnf("Failed to query the Pull Request status for %s %s", pullRequestInfo.Link, err)
			} else {
				elaspedString := time.Since(start).String()
				if pr.Merged {
					if pr.MergeSha == "" {
						log.Logger().Infof("Pull Request %s was merged but we didn't yet have a merge SHA after waiting %s", termcolor.ColorInfo(pr.Link), elaspedString)
						return nil
					}
					log.Logger().Infof("Pull Request %s was merged at sha %s after waiting %s", termcolor.ColorInfo(pr.Link), termcolor.ColorInfo(pr.MergeSha), elaspedString)
					return nil
				} else if pr.Closed {
					log.Logger().Warnf("Pull Request %s is closed after waiting %s", termcolor.ColorInfo(pr.Link), elaspedString)
					return nil
				}
			}
			if time.Now().After(end) {
				return fmt.Errorf("timed out waiting for pull request %s to merge. Waited %s", pr.Link, durationString)
			}
			time.Sleep(o.PullRequestPollPeriod)
		}
	}
	return nil
}

func (o *ImportOptions) IsGitHubAppMode() (bool, error) {
	return false, nil
}

func (o *ImportOptions) defaultGitServerURLFromDevEnv() (string, error) {
	gitURL := ""
	if o.DevEnv != nil {
		gitURL = o.DevEnv.Spec.Source.URL
	}
	if gitURL == "" {
		// let's default to github
		return giturl.GitHubURL, nil
	}
	gitInfo, err := giturl.ParseGitURL(gitURL)
	if err != nil {
		return "", errors.Wrapf(err, "failed to parse git URL %s", gitURL)
	}
	return gitInfo.HostURL(), nil
}
