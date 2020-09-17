package root

import (
	"os"
	"strconv"

	"github.com/jenkins-x/jx-logging/pkg/log"
	"github.com/spf13/cobra"
)

func setLoggingLevel(cmd *cobra.Command) {
	verbose := false
	flag := cmd.Flag("verbose")
	if flag != nil {
		var err error
		verbose, err = strconv.ParseBool(flag.Value.String())
		if err != nil {
			log.Logger().Errorf("Unable to check if the verbose flag is set")
		}
	}

	level := os.Getenv("JX_LOG_LEVEL")
	if level != "" {
		if verbose {
			log.Logger().Trace("The JX_LOG_LEVEL environment variable took precedence over the verbose flag")
		}

		err := log.SetLevel(level)
		if err != nil {
			log.Logger().Errorf("Unable to set log level to %s", level)
		}
	} else {
		if verbose {
			err := log.SetLevel("debug")
			if err != nil {
				log.Logger().Errorf("Unable to set log level to debug")
			}
		} else {
			err := log.SetLevel("info")
			if err != nil {
				log.Logger().Errorf("Unable to set log level to info")
			}
		}
	}
}
