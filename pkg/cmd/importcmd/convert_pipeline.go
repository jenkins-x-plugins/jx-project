package importcmd

import (
	"os"
	"path/filepath"

	"github.com/jenkins-x/jx-helpers/v3/pkg/cmdrunner"
	"github.com/pkg/errors"
)

// convertOldPipeline converts the old pipeline and returns true if it was converted
func (o *ImportOptions) convertOldPipeline() (bool, error) {
	if !o.BatchMode {
		flag, err := o.Input.Confirm("Convert the old jenkins-x.yml file to the new tekton yaml?: ", true, "please confirm you wish to convert the pipeline")
		if err != nil {
			return flag, errors.Wrapf(err, "failed to conform conversion")
		}
		if !flag {
			return false, nil
		}
	}
	dir, err := filepath.Abs(o.Dir)
	if err != nil {
		return false, errors.Wrapf(err, "failed to get absolute dir of %s", o.Dir)
	}

	args := []string{"pipeline", "convert", "--dir", dir}
	if o.BatchMode {
		args = append(args, "--batch-mode")
	}
	c := &cmdrunner.Command{
		Dir:  dir,
		Name: "jx",
		Args: args,
		Out:  os.Stdout,
		Err:  os.Stderr,
		In:   os.Stdin,
	}
	_, err = o.CommandRunner(c)
	if err != nil {
		return false, errors.Wrapf(err, "failed to convert old pipeline via %s", c.CLI())
	}
	return true, nil
}
