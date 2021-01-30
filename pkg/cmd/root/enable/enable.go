package enable

import (
	"github.com/jenkins-x/jx-helpers/v3/pkg/gitclient"
	"github.com/pkg/errors"
	"io"

	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/templates"
	"github.com/jenkins-x/jx-helpers/v3/pkg/termcolor"
	"github.com/jenkins-x/jx-project/pkg/cmd/importcmd"
	"github.com/jenkins-x/lighthouse-client/pkg/triggerconfig"
	"github.com/spf13/cobra"
	tektonv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"strings"
)

// Options contains the command line options
type Options struct {
	importcmd.ImportOptions
}

var (
	info = termcolor.ColorInfo

	cmdLong = templates.LongDesc(`
		Enables lighthouse pipelines in the current directory
`)

	cmdExample = templates.Examples(`
		# Enables lighthouse pipelines in the current dir
		jx project enable
	`)
)

// Trigger the found trigger configs
type Trigger struct {
	Path      string
	Config    *triggerconfig.Config
	Names     []string
	Pipelines map[string]*tektonv1beta1.PipelineRun
}

// NewCmdPipelineEnable creates the command
func NewCmdPipelineEnable() (*cobra.Command, *Options) {
	o := &Options{}

	o.ImportOptions.NoDevPullRequest = true
	o.ImportOptions.DisableStartPipeline = true
	o.ImportOptions.DisableStartPipeline = true
	o.ImportOptions.IgnoreJenkinsXFile = true

	o.ImportOptions.PackFilter = func(pack *importcmd.Pack) {
		// lets exclude everything from the pack other than lighthouse files
		m := map[string]io.ReadCloser{}
		for k, v := range pack.Files {
			if strings.HasPrefix(k, ".lighthouse") {
				m[k] = v
			}
		}
		pack.Charts = nil
		pack.Files = m
	}

	cmd := &cobra.Command{
		Use:     "enable",
		Short:   "Enables lighthouse pipelines in the current directory",
		Long:    cmdLong,
		Example: cmdExample,
		Aliases: []string{"dump"},
		Run: func(cmd *cobra.Command, args []string) {
			err := o.Run()
			helper.CheckErr(err)
		},
	}

	cmd.Flags().StringVarP(&o.Dir, "dir", "", ".", "Specify the directory to import")
	cmd.Flags().StringVarP(&o.Pack, "pack", "", "", "The name of the pipeline catalog pack to use. If none is specified it will be chosen based on matching the source code languages")

	o.BaseOptions.AddBaseFlags(cmd)
	o.ScmFactory.AddFlags(cmd)

	return cmd, o

}

// Validate verifies settings
func (o *Options) Validate() error {
	return nil
}

// Run implements this command
func (o *Options) Run() error {
	err := o.ImportOptions.Run()
	if err != nil {
		return errors.Wrapf(err, "failed to enable lighthouse pipelines")
	}

	err = gitclient.Add(o.Git(), o.Dir, ".lighthouse")
	if err != nil {
		return errors.Wrapf(err, "failed to add the .lighthouse dir to git")
	}
	return nil
}
