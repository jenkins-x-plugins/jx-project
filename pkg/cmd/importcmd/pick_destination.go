package importcmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/jenkins-x/jx-gitops/pkg/sourceconfigs"
	"github.com/jenkins-x/jx-helpers/v3/pkg/stringhelpers"
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
	Server  string
	Enabled bool
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
func (o *ImportOptions) PickImportDestination(devEnvCloneDir, jenkinsfile string) (ImportDestination, error) {
	sourceConfig, err := sourceconfigs.LoadSourceConfig(devEnvCloneDir, true)
	if err != nil {
		return o.Destination, errors.Wrapf(err, "failed to load the source config")
	}

	// lets check CLI arguments to pick the destination
	if o.Destination.JenkinsX.Enabled {
		return o.Destination, nil
	}
	if o.Destination.JenkinsfileRunner.Image != "" {
		o.Destination.JenkinsfileRunner.Enabled = true
		o.Destination.JenkinsX.Enabled = true
		return o.Destination, nil
	}
	if o.Destination.Jenkins.Server != "" {
		o.Destination.Jenkins.Enabled = true
		return o.Destination, nil
	}

	// discover the jenkins servers from the source config
	var names []string
	for _, jc := range sourceConfig.Spec.JenkinsServers {
		if jc.Server == "" {
			continue
		}
		names = append(names, jc.Server)
	}
	if o.Destination.Jenkins.Server != "" && stringhelpers.StringArrayIndex(names, o.Destination.Jenkins.Server) < 0 {
		names = append(names, o.Destination.Jenkins.Server)
	}
	sort.Strings(names)

	if len(names) == 0 {
		o.Destination.JenkinsX.Enabled = true
		return o.Destination, nil
	}

	if o.BatchMode {
		if o.Destination.JenkinsfileRunner.Enabled {
			o.Destination.JenkinsfileRunner = JenkinsfileRunnerDestination{Enabled: true}
			return o.Destination, nil
		}
		if o.Destination.Jenkins.Enabled {
			if len(names) == 1 {
				o.Destination.Jenkins.Server = names[0]
				o.Destination.Jenkins.Enabled = true
				return o.Destination, nil
			}
		}
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
				Server:  name,
				Enabled: true,
			},
		}
	}

	if jenkinsfile != "" {
		text := "Jenkinsfile runner"
		actionChoices = append(actionChoices, text)
		actions[text] = ImportDestination{JenkinsX: JenkinsXDestination{Enabled: true}, JenkinsfileRunner: JenkinsfileRunnerDestination{Enabled: true}}
	}

	name, err := o.Input.PickNameWithDefault(actionChoices, "Where would you like to import this project to?",
		"", "you can import into Jenkins X and use cloud native pipelines with Tekton or import in a Jenkins server")
	if err != nil {
		return o.Destination, err
	}
	if name == "" {
		return o.Destination, fmt.Errorf("nothing chosen")
	}
	o.Destination = actions[name]
	return o.Destination, nil
}
