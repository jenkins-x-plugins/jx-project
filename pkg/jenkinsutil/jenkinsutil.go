package jenkinsutil

import (
	"github.com/spf13/cobra"
)

// JenkinsSelectorOptions used to represent the options used to refer to a Jenkins.
// if nothing is specified it assumes the current team is using a static Jenkins server as its execution engine.
// otherwise we can refer to other additional Jenkins Apps to implement custom Jenkins servers
type JenkinsSelectorOptions struct {
	// JenkinsName the name of the Jenkins Operator Service for HTTP to use
	JenkinsName string

	// Selector label selector to find the Jenkins Operator Services
	Selector string

	// NameLabel label the label to find the name of the Jenkins service
	NameLabel string

	// DevelopmentJenkinsURL a local URL to use to talk to the jenkins server if the servers do not have Ingress
	// and you want to test out using the jenkins client locally
	DevelopmentJenkinsURL string
}

// AddFlags add the command flags for picking a custom Jenkins App to work with
func (o *JenkinsSelectorOptions) AddFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&o.JenkinsName, "jenkins", "", "", "The name of the Jenkin server provisioned by the Jenkins Operator")
	cmd.Flags().StringVarP(&o.Selector, "selector", "", JenkinsSelector, "The kubernetes label selector to find the Jenkins Operator Services for Jenkins HTTP servers")
	cmd.Flags().StringVarP(&o.NameLabel, "name-label", "", JenkinsNameLabel, "The kubernetes label used to specify the Jenkins service name")
}
