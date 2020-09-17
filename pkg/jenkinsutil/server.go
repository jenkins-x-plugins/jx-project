package jenkinsutil

// JenkinsServer represents a jenkins server discovered via Service selectors or via the
// trigger-pipeline secrets
type JenkinsServer struct {
	// Name the name of the Jenkins server in the registry. Should be a valid kubernetes name
	Name string

	// URL the URL to connect to the Jenkins server
	URL string

	// SecretName the name of the Secret in the registry
	SecretName string
}
