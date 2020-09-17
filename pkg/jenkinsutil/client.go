package jenkinsutil

type ClientFactory struct {
	Namespace             string
	Batch                 bool
	InCluster             bool
	DevelopmentJenkinsURL string
}
