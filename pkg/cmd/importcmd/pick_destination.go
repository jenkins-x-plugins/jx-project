package importcmd

import (
	"fmt"
	"strings"

	"github.com/jenkins-x-labs/trigger-pipeline/pkg/jenkinsutil"
	"github.com/jenkins-x/jx/v2/pkg/util"
	"github.com/pkg/errors"
)

// Destination where do we want to import the project to
type ImportDestination struct {
	JenkinsX          JenkinsXDestination
	Jenkins           JenkinsDestination
	JenkinsfileRunner JenkinsfileRunnerDestination
}

// JenkinsXDestination configures how to import to Jenkins X
type JenkinsXDestination struct {
	Enabled bool
}

// JenkinsDestination configures how to import to a Jenkins server
type JenkinsDestination struct {
	jenkinsutil.JenkinsSelectorOptions
	Enabled             bool
	JenkinsServiceNames []string
}

// JenkinsfileRunnerDestination configures how to import to JenkinfilesRunner
type JenkinsfileRunnerDestination struct {
	Enabled bool
	Image   string
}

const (
	jenkinsXDestination = "Jenkins X"
)

// PickImportDestination picks where to import the project to
func (o *ImportOptions) PickImportDestination(cf *jenkinsutil.ClientFactory, jenkinsfile string) (ImportDestination, error) {
	// lets check CLI arguments to pick the destination
	if o.Destination.JenkinsX.Enabled {
		return o.Destination, nil
	}
	if o.Destination.JenkinsfileRunner.Image != "" {
		o.Destination.JenkinsfileRunner.Enabled = true
		o.Destination.JenkinsX.Enabled = true
		return o.Destination, nil
	}
	if o.Destination.Jenkins.JenkinsName != "" {
		o.Destination.Jenkins.Enabled = true
		return o.Destination, nil
	}

	handles := o.CommonOptions.GetIOFileHandles()

	names := o.Destination.Jenkins.JenkinsServiceNames
	if len(names) == 0 {
		var err error
		_, names, err = jenkinsutil.FindJenkinsServers(cf, &o.Destination.Jenkins.JenkinsSelectorOptions)
		if err != nil {
			return o.Destination, errors.Wrapf(err, "failed to find Jenkins service names")
		}
		o.Destination.Jenkins.JenkinsServiceNames = names
	}

	if len(names) == 0 {
		o.Destination.JenkinsX.Enabled = true
		return o.Destination, nil
	}

	if o.BatchMode {
		return o.Destination, fmt.Errorf("no import destination specified in batch mode. Please specify --jenkins or --jx")
	}
	// lets add a list of choices...
	actionChoices := []string{jenkinsXDestination}
	actions := map[string]ImportDestination{
		jenkinsXDestination: {JenkinsX: JenkinsXDestination{Enabled: true}},
	}
	for _, name := range names {
		text := fmt.Sprintf("Jenkins: %s", strings.TrimPrefix(name, "jenkins-operator-http-"))
		actionChoices = append(actionChoices, text)
		actions[text] = ImportDestination{
			Jenkins: JenkinsDestination{
				JenkinsSelectorOptions: jenkinsutil.JenkinsSelectorOptions{JenkinsName: name},
				Enabled:                true,
			},
		}
	}

	if jenkinsfile != "" {
		text := "Jenkinsfile runner"
		actionChoices = append(actionChoices, text)
		actions[text] = ImportDestination{JenkinsX: JenkinsXDestination{Enabled: true}, JenkinsfileRunner: JenkinsfileRunnerDestination{Enabled: true}}
	}

	name, err := util.PickName(actionChoices, "Where would you like to import this project to?",
		"you can import into Jenkins X and use cloud native pipelines with Tekton or import in a Jenkins server", handles)
	if err != nil {
		return o.Destination, err
	}
	if name == "" {
		return o.Destination, fmt.Errorf("nothing chosen")
	}
	o.Destination = actions[name]
	return o.Destination, nil
}
