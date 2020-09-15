package importcmd

const (
	// PlaceHolderPrefix is prefix for placeholders
	PlaceHolderPrefix = "REPLACE_ME"

	// PlaceHolderAppName placeholder for app name
	PlaceHolderAppName = PlaceHolderPrefix + "_APP_NAME"

	// PlaceHolderGitProvider placeholder for git provider
	PlaceHolderGitProvider = PlaceHolderPrefix + "_GIT_PROVIDER"

	// PlaceHolderOrg placeholder for org
	PlaceHolderOrg = PlaceHolderPrefix + "_ORG"

	// PlaceHolderDockerRegistryOrg placeholder for docker registry
	PlaceHolderDockerRegistryOrg = PlaceHolderPrefix + "_DOCKER_REGISTRY_ORG"

	MinimumMavenDeployVersion = "2.8.2"

	// DeployKindKnative for knative serve based deployments
	DeployKindKnative = "knative"

	// DeployKindDefault for default kubernetes Deployment + Service deployment kinds
	DeployKindDefault = "default"

	// OptionKind to specify the kind of something (such as the kind of a deployment)
	OptionKind = "kind"

	// OptionCanary should we enable canary rollouts (progressive delivery)
	OptionCanary = "canary"

	// OptionHPA should we enable horizontal pod autoscaler for deployments
	OptionHPA = "hpa"

	DefaultGitIgnoreFile = `
.project
.classpath
.idea
.cache
.DS_Store
*.im?
target
work
`
)
