// +build !windows

package app

import (
	"os"

	"github.com/jenkins-x/jx-project/pkg/cmd/create"
	"github.com/jenkins-x/jx/pkg/cmd/clients"
	"github.com/jenkins-x/jx/pkg/cmd/opts"
)

// Run runs the command, if args are not nil they will be set on the command
func Run(args []string) error {
	f := clients.NewFactory()
	commonOptions := opts.NewCommonOptionsWithTerm(f, os.Stdin, os.Stdout, os.Stderr)
	cmd := create.NewCmdCreateProject(commonOptions)
	if args != nil {
		args = args[1:]
		cmd.SetArgs(args)
	}
	return cmd.Execute()
}
