package root

import (
	"fmt"
	"os"

	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/v3/pkg/homedir"
	"github.com/jenkins-x/jx-helpers/v3/pkg/stringhelpers"
	"github.com/jenkins-x/jx-helpers/v3/pkg/termcolor"
	"github.com/jenkins-x/jx-project/pkg/cmd/common"
	"github.com/jenkins-x/jx-project/pkg/cmd/importcmd"
	"github.com/pkg/errors"

	"github.com/spf13/cobra"

	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/templates"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/jenkins-x/jx-project/pkg/spring"
)

var (
	createSpringLong = templates.LongDesc(`
		Creates a new Spring Boot application and then optionally setups CI/CD pipelines and GitOps promotion.

		You can see a demo of this command here: [https://jenkins-x.io/demos/create_spring/](https://jenkins-x.io/demos/create_spring/)

		For more documentation see: [https://jenkins-x.io/developing/create-spring/](https://jenkins-x.io/developing/create-spring/)

` + helper.SeeAlsoText("jx create project"))

	createSpringExample = templates.Examples(`
		# Create a Spring Boot application where you use the terminal to pick the values
		%s spring

		# Creates a Spring Boot application passing in the required dependencies
		%s spring -d web -d actuator

		# To pick the advanced options (such as what package type maven-project/gradle-project) etc then use
		%s spring -x

		# To create a gradle project use:
		%s spring --type gradle-project
	`)
)

// CreateSpringOptions the options for the create spring command
type CreateSpringOptions struct {
	Options

	Advanced   bool
	SpringForm spring.SpringBootForm
}

// NewCmdCreateSpring creates a command object for the "create" command
func NewCmdCreateSpring() *cobra.Command {
	options := &CreateSpringOptions{}

	cmd := &cobra.Command{
		Use:     "spring",
		Short:   "Create a new Spring Boot application and import the generated code into Git and Jenkins for CI/CD",
		Long:    createSpringLong,
		Example: fmt.Sprintf(createSpringExample, common.BinaryName, common.BinaryName, common.BinaryName, common.BinaryName),
		Run: func(cmd *cobra.Command, args []string) {
			options.Args = args
			err := options.Run()
			helper.CheckErr(err)
		},
	}
	options.addCreateAppFlags(cmd)

	cmd.Flags().BoolVarP(&options.Advanced, "advanced", "x", false, "Advanced mode can show more detailed forms for some resource kinds like springboot")

	cmd.Flags().StringArrayVarP(&options.SpringForm.DependencyKinds, spring.OptionDependencyKind, "k", spring.DefaultDependencyKinds, "Default dependency kinds to choose from")
	cmd.Flags().StringArrayVarP(&options.SpringForm.Dependencies, spring.OptionDependency, "d", []string{}, "Spring Boot dependencies")
	cmd.Flags().StringVarP(&options.SpringForm.GroupId, spring.OptionGroupId, "g", "", "Group ID to generate")
	cmd.Flags().StringVarP(&options.SpringForm.ArtifactId, spring.OptionArtifactId, "a", "", "Artifact ID to generate")
	cmd.Flags().StringVarP(&options.SpringForm.Language, spring.OptionLanguage, "l", "", "Language to generate")
	cmd.Flags().StringVarP(&options.SpringForm.BootVersion, spring.OptionBootVersion, "t", "", "Spring Boot version")
	cmd.Flags().StringVarP(&options.SpringForm.JavaVersion, spring.OptionJavaVersion, "j", "", "Java version")
	cmd.Flags().StringVarP(&options.SpringForm.Packaging, spring.OptionPackaging, "p", "", "Packaging")
	cmd.Flags().StringVarP(&options.SpringForm.Type, spring.OptionType, "", "", "Project Type (such as maven-project or gradle-project)")

	return cmd
}

// Run implements the command
func (o *CreateSpringOptions) Run() error {
	err := o.Validate()
	if err != nil {
		return errors.Wrapf(err, "failed to validate options")
	}

	cacheDir, err := homedir.CacheDir(os.Getenv("JX3_HOME"), ".jx3")
	if err != nil {
		return err
	}

	data := &o.SpringForm

	var details *importcmd.CreateRepoData

	if !o.BatchMode {
		details, err = o.GetGitRepositoryDetails()
		if err != nil {
			return err
		}

		data.ArtifactId = details.RepoName
	}

	model, err := spring.LoadSpringBoot(cacheDir)
	if err != nil {
		return fmt.Errorf("Failed to load Spring Boot model %s", err)
	}
	err = model.CreateSurvey(&o.SpringForm, o.Advanced, o.BatchMode)
	if err != nil {
		return err
	}

	// always add in actuator as its required for health checking
	if stringhelpers.StringArrayIndex(o.SpringForm.Dependencies, "actuator") < 0 {
		o.SpringForm.Dependencies = append(o.SpringForm.Dependencies, "actuator")
	}
	// always add web as the JVM tends to terminate if its not added
	if stringhelpers.StringArrayIndex(o.SpringForm.Dependencies, "web") < 0 {
		o.SpringForm.Dependencies = append(o.SpringForm.Dependencies, "web")
	}

	dir := o.OutDir
	if dir == "" {
		dir, err = os.Getwd()
		if err != nil {
			return err
		}
	}

	outDir, err := data.CreateProject(dir)
	if err != nil {
		return err
	}
	log.Logger().Infof("Created Spring Boot project at %s", termcolor.ColorInfo(outDir))

	if details != nil {
		o.ConfigureImportOptions(details)
	}

	return o.ImportCreatedProject(outDir)
}
