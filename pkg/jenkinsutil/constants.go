package jenkinsutil

const (
	// TriggerJenkinsServerEnv the environment variable used to choose the jenkins server to trigger
	TriggerJenkinsServerEnv = "TRIGGER_JENKINS_SERVER"

	// JenkinsSelector the default selector to find Jenkins services as Kubernetes Services
	JenkinsSelector = "app=jenkins-operator"

	// JenkinsNameLabel default label to indicate the name of the Jenkins service
	JenkinsNameLabel = "jenkins-cr"
)
