// +build !windows

package app

import (
	"github.com/jenkins-x-plugins/jx-project/pkg/cmd/root"
)

// Run runs the command, if args are not nil they will be set on the command
func Run(args []string) error {
	cmd, _ := root.NewCmdMain()
	if args != nil {
		args = args[1:]
		cmd.SetArgs(args)
	}
	return cmd.Execute()
}
