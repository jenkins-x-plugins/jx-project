package root

import (
	"fmt"
	"os"

	"github.com/jenkins-x/jx-project/pkg/cmd/common"
	"github.com/jenkins-x/jx-project/pkg/cmd/importcmd"
	"github.com/jenkins-x/jx-project/pkg/cmd/root/pullrequest"
	"github.com/jenkins-x/jx/v2/pkg/cmd/clients"
	"github.com/jenkins-x/jx/v2/pkg/cmd/create/options"
	"github.com/jenkins-x/jx/v2/pkg/helm"

	"github.com/jenkins-x/jx/v2/pkg/cmd/helper"

	"github.com/jenkins-x/jx/v2/pkg/cmd/opts"
	"github.com/jenkins-x/jx/v2/pkg/cmd/templates"
	"github.com/jenkins-x/jx/v2/pkg/util"
	"github.com/spf13/cobra"
)

const (
	createQuickstartName = "Create new application from a Quickstart"
	createSpringName     = "Create new Spring Boot microservice"
	importDirName        = "Import existing code from a directory"
	importGitName        = "Import code from a git repository"
	importGitHubName     = "Import code from a github repository"
)

var (
	createProjectNames = []string{
		createQuickstartName,
		createSpringName,
		importDirName,
		importGitName,
		importGitHubName,
	}

	createProjectLong = templates.LongDesc(`
		Create a new project by importing code, creating a quickstart or custom wizard for spring.

`)

	createProjectExample = templates.Examples(`
		# Create a project using the wizard
		%s
	`)
)

// Options contains the command line options
type Options struct {
	importcmd.ImportOptions

	OutDir             string
	DisableImport      bool
	GithubAppInstalled bool
}

// WizardOptions the options for the command
type WizardOptions struct {
	options.CreateOptions
}

// NewCmdMain creates a command object for the command
func NewCmdMain() (*cobra.Command, *WizardOptions) {
	f := clients.NewFactory()
	commonOptions := opts.NewCommonOptionsWithTerm(f, os.Stdin, os.Stdout, os.Stderr)
	commonOptions.SetHelm(helm.NewHelmCLI("helm", helm.V3, "", false))
	return NewCmdMainWithOptions(commonOptions)
}

// NewCmdMainWithOptions creates a command object for the command
func NewCmdMainWithOptions(commonOpts *opts.CommonOptions) (*cobra.Command, *WizardOptions) {
	options := &WizardOptions{
		CreateOptions: options.CreateOptions{
			CommonOptions: commonOpts,
		},
	}
	cmd := &cobra.Command{
		Use:     "project",
		Short:   "Create a new project by importing code, creating a quickstart or custom wizard for spring",
		Long:    createProjectLong,
		Example: fmt.Sprintf(createProjectExample, common.BinaryName),
		Run: func(cmd *cobra.Command, args []string) {
			setLoggingLevel(cmd)
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			helper.CheckErr(err)
		},
	}

	cmd.AddCommand(NewCmdCreateQuickstart(commonOpts))
	cmd.AddCommand(NewCmdCreateSpring(commonOpts))
	cmd.AddCommand(importcmd.NewCmdImport(commonOpts))
	cmd.AddCommand(pullrequest.NewCmdCreatePullRequest(commonOpts))

	return cmd, options
}

// Run implements the command
func (o *WizardOptions) Run() error {
	name, err := util.PickName(createProjectNames, "Which kind of project you want to create: ",
		"there are a number of different wizards for creating or importing new projects.",
		o.GetIOFileHandles())
	if err != nil {
		return err
	}
	switch name {
	case createQuickstartName:
		return o.createQuickstart()
	case createSpringName:
		return o.createSpring()
	case importDirName:
		return o.importDir()
	case importGitName:
		return o.importGit()
	case importGitHubName:
		return o.importGithubProject()
	default:
		return fmt.Errorf("Unknown selection: %s\n", name)
	}
}

func (o *WizardOptions) createQuickstart() error {
	w := &CreateQuickstartOptions{}
	w.CommonOptions = o.CommonOptions
	return w.Run()
}

func (o *WizardOptions) createSpring() error {
	w := &CreateSpringOptions{}
	w.CommonOptions = o.CommonOptions
	return w.Run()
}

func (o *WizardOptions) importDir() error {
	wd, err := os.Getwd()
	if err != nil {
		return err
	}
	dir, err := util.PickValue("Which directory contains the source code: ", wd, true,
		"Please specify the directory which contains the source code you want to use for your new project", o.GetIOFileHandles())
	if err != nil {
		return err
	}
	w := &importcmd.ImportOptions{
		Dir: dir,
	}
	w.CommonOptions = o.CommonOptions
	return w.Run()
}

func (o *WizardOptions) importGit() error {
	repoUrl, err := util.PickValue("Which git repository URL to import: ", "", true,
		"Please specify the git URL which contains the source code you want to use for your new project", o.GetIOFileHandles())
	if err != nil {
		return err
	}

	w := &importcmd.ImportOptions{
		RepoURL: repoUrl,
	}
	w.CommonOptions = o.CommonOptions
	return w.Run()
}

func (o *WizardOptions) importGithubProject() error {
	w := &importcmd.ImportOptions{
		GitHub: true,
	}
	w.CommonOptions = o.CommonOptions
	return w.Run()
}

// DoImport imports the project created at the given directory
func (o *Options) ImportCreatedProject(outDir string) error {
	if o.DisableImport {
		return nil
	}
	importOptions := &o.ImportOptions
	importOptions.Dir = outDir
	importOptions.DisableDotGitSearch = true
	importOptions.GithubAppInstalled = o.GithubAppInstalled
	return importOptions.Run()
}

func (o *Options) addCreateAppFlags(cmd *cobra.Command) {
	cmd.Flags().BoolVarP(&o.DisableImport, "no-import", "", false, "Disable import after the creation")
	cmd.Flags().StringVarP(&o.OutDir, opts.OptionOutputDir, "o", "", "Directory to output the project to. Defaults to the current directory")

	o.AddImportFlags(cmd, true)
}
