package enable

import (
	"io"
	"strings"

	"github.com/jenkins-x/jx-helpers/v3/pkg/gitclient"
	"github.com/pkg/errors"

	"github.com/jenkins-x-plugins/jx-project/pkg/cmd/importcmd"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/templates"
	"github.com/spf13/cobra"
)

// Options contains the command line options
type Options struct {
	importcmd.ImportOptions
	RegenCharts bool
}

var (
	cmdLong = templates.LongDesc(`
		Enables lighthouse pipelines in the current directory
`)

	cmdExample = templates.Examples(`
		# Enables lighthouse pipelines in the current dir
		jx project enable
	`)
)

// NewCmdPipelineEnable creates the command
func NewCmdPipelineEnable() (*cobra.Command, *Options) {
	o := &Options{}

	o.ImportOptions.NoDevPullRequest = true
	o.ImportOptions.DisableStartPipeline = true
	o.ImportOptions.DisableStartPipeline = true

	if o.RegenCharts {
		o.ImportOptions.PackFilter = func(pack *importcmd.Pack) {
			// let's exclude everything from the pack other than lighthouse files
			m := map[string]io.ReadCloser{}
			for k, v := range pack.Files {
				if strings.HasPrefix(k, ".lighthouse") {
					m[k] = v
				}
			}
			pack.Charts = nil
			pack.Files = m
		}
	}

	cmd := &cobra.Command{
		Use:     "enable",
		Short:   "Enables lighthouse pipelines in the current directory",
		Long:    cmdLong,
		Example: cmdExample,
		Aliases: []string{"dump"},
		Run: func(_ *cobra.Command, _ []string) {
			err := o.Run()
			helper.CheckErr(err)
		},
	}

	cmd.Flags().BoolVarP(&o.RegenCharts, "charts", "", false, "Should we regen the charts")
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
