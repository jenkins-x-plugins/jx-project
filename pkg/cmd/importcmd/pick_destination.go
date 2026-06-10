package importcmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/jenkins-x-plugins/jx-gitops/pkg/sourceconfigs"
	"github.com/jenkins-x/jx-helpers/v3/pkg/kube/naming"
	"github.com/jenkins-x/jx-helpers/v3/pkg/stringhelpers"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
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
	jenkinsXDestination = "Jenkins X automated cloud native pipelines via Tekton"
)

// PickImportDestination picks where to import the project to
func (o *ImportOptions) PickImportDestination(devEnvCloneDir string) (ImportDestination, error) {
	sourceConfig, err := sourceconfigs.LoadSourceConfig(devEnvCloneDir, true)
	if err != nil {
		return o.Destination, errors.Wrapf(err, "failed to load the source config")
	}

	// let's check CLI arguments to pick the destination
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

	log.Logger().Info("")
	log.Logger().Infof("this project has a %s so lets choose how you want to setup CI", info("Jenkinsfile"))
	log.Logger().Info("")

	if len(names) == 0 {
		log.Logger().Info("there are currently no Jenkins servers configured in your cluster git repository")
		log.Logger().Info("")

		flag, err := o.Input.Confirm("Would you like to add a Jenkins server?: ", true, "There is configured jenkins server. Please confirm if you would like to add a new server otherwise we will use the Jenkinsfile runner")
		if err != nil {
			return o.Destination, errors.Wrapf(err, "failed to get the confirm flag")
		}
		if flag {
			name, err := o.Input.PickValue("Name of the new jenkins server: ", "myjenkins", true, "please enter the name of the new jenkins server. Should be usable inside a DNS name so be lowercase starting with a letter")
			if err != nil {
				return o.Destination, errors.Wrapf(err, "failed to enter the name of the jenkins server")
			}
			name = strings.TrimSpace(name)
			if name == "" {
				return o.Destination, errors.Errorf("no jenkins server name entered")
			}
			o.Destination.Jenkins.Server = naming.ToValidName(name)
			o.Destination.Jenkins.Enabled = true
			return o.Destination, nil
		}
	}

	// let's add a list of choices...
	actionChoices := []string{jenkinsXDestination}
	actions := map[string]ImportDestination{
		jenkinsXDestination: {JenkinsX: JenkinsXDestination{Enabled: true}},
	}
	for _, name := range names {
		text := fmt.Sprintf("Jenkins pipelines on server: %s", strings.TrimPrefix(name, "jenkins-operator-http-"))
		actionChoices = append(actionChoices, text)
		actions[text] = ImportDestination{
			Jenkins: JenkinsDestination{
				Server:  name,
				Enabled: true,
			},
		}
	}

	// add jenkinsfile runner as an option
	text := "Jenkinsfile runner"
	actionChoices = append(actionChoices, text)
	actions[text] = ImportDestination{JenkinsX: JenkinsXDestination{Enabled: true}, JenkinsfileRunner: JenkinsfileRunnerDestination{Enabled: true}}

	name, err := o.Input.PickNameWithDefault(actionChoices, "How would you like to import this project?",
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
