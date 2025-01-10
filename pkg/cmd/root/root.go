package root

import (
	"fmt"
	"os"

	"github.com/jenkins-x-plugins/jx-project/pkg/cmd/root/enable"

	"github.com/jenkins-x-plugins/jx-project/pkg/cmd/common"
	"github.com/jenkins-x-plugins/jx-project/pkg/cmd/importcmd"
	"github.com/jenkins-x-plugins/jx-project/pkg/cmd/root/pullrequest"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/v3/pkg/input"
	"github.com/jenkins-x/jx-helpers/v3/pkg/input/survey"

	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/templates"
	"github.com/spf13/cobra"
)

const (
	createQuickstartName   = "Create new application from a Quickstart"
	createMLQuickstartName = "Create new application from a Machine Learning Quickstart"
	createSpringName       = "Create new Spring Boot microservice"
	importDirName          = "Import existing code from a directory"
	importGitName          = "Import code from a git repository"
	importGitHubName       = "Import code from a github repository"
)

var (
	createProjectNames = []string{
		createQuickstartName,
		createMLQuickstartName,
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
	Input input.Interface
}

// NewCmdMain creates a command object for the command
func NewCmdMain() (*cobra.Command, *WizardOptions) {
	options := &WizardOptions{}
	cmd := &cobra.Command{
		Use:     common.BinaryName,
		Short:   "Create a new project by importing code, creating a quickstart or custom wizard for spring",
		Long:    createProjectLong,
		Example: fmt.Sprintf(createProjectExample, common.BinaryName),
		Run: func(cmd *cobra.Command, _ []string) {
			setLoggingLevel(cmd)
			err := options.Run()
			helper.CheckErr(err)
		},
	}

	cmd.AddCommand(cobras.SplitCommand(enable.NewCmdPipelineEnable()))
	cmd.AddCommand(cobras.SplitCommand(NewCmdCreateQuickstart()))
	cmd.AddCommand(cobras.SplitCommand(NewCmdCreateMLQuickstart()))
	cmd.AddCommand(NewCmdCreateSpring())
	cmd.AddCommand(importcmd.NewCmdImport())
	cmd.AddCommand(pullrequest.NewCmdCreatePullRequest())

	return cmd, options
}

// Run implements the command
func (o *WizardOptions) Run() error {
	if o.Input == nil {
		o.Input = survey.NewInput()
	}

	name, err := o.Input.PickNameWithDefault(createProjectNames, "Which kind of project you want to create: ",
		"", "there are a number of different wizards for creating or importing new projects.")
	if err != nil {
		return err
	}
	switch name {
	case createQuickstartName:
		return o.createQuickstart()
	case createMLQuickstartName:
		return o.createMLQuickstart()
	case createSpringName:
		return o.createSpring()
	case importDirName:
		return o.importDir()
	case importGitName:
		return o.importGit()
	case importGitHubName:
		return o.importGithubProject()
	default:
		return fmt.Errorf("unknown selection: %s", name)
	}
}

func (o *WizardOptions) createQuickstart() error {
	w := &CreateQuickstartOptions{}
	return w.Run()
}

func (o *WizardOptions) createMLQuickstart() error {
	w := &CreateMLQuickstartOptions{}
	return w.Run()
}

func (o *WizardOptions) createSpring() error {
	w := &CreateSpringOptions{}
	return w.Run()
}

func (o *WizardOptions) importDir() error {
	wd, err := os.Getwd()
	if err != nil {
		return err
	}
	dir, err := o.Input.PickValue("Which directory contains the source code: ", wd, true,
		"Please specify the directory which contains the source code you want to use for your new project")
	if err != nil {
		return err
	}
	w := &importcmd.ImportOptions{
		Dir: dir,
	}
	return w.Run()
}

func (o *WizardOptions) importGit() error {
	repoURL, err := o.Input.PickValue("Which git repository URL to import: ", "", true,
		"Please specify the git URL which contains the source code you want to use for your new project")
	if err != nil {
		return err
	}

	w := &importcmd.ImportOptions{
		RepoURL: repoURL,
	}
	return w.Run()
}

func (o *WizardOptions) importGithubProject() error {
	w := &importcmd.ImportOptions{
		GitHub: true,
	}
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
	cmd.Flags().StringVarP(&o.OutDir, "output-dir", "o", "", "Directory to output the project to. Defaults to the current directory")

	o.AddImportFlags(cmd, true)
}
