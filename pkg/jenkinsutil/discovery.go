package jenkinsutil

var (
	// DefaultJenkinsSelector default selector options to use if finding Jenkins services
	DefaultJenkinsSelector = JenkinsSelectorOptions{
		Selector:  JenkinsSelector,
		NameLabel: JenkinsNameLabel,
	}
)

// FindJenkinsServers discovers the jenkins services
func FindJenkinsServers(f *ClientFactory, jenkinsSelector *JenkinsSelectorOptions) (map[string]*JenkinsServer, []string, error) {
	return nil, nil, nil
}
