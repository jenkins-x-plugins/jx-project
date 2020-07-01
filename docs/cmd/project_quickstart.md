## project quickstart

Create a new app from a Quickstart and import the generated code into Git and Jenkins for CI/CD

***Aliases**: arch*

### Usage

```
project quickstart
```

### Synopsis

Create a new project from a sample/starter (found in https://github.com/jenkins-x-quickstarts) 

This will create a new project for you from the selected template. It will exclude any work-in-progress repos (containing the "WIP-" pattern) 

For more documentation see: https://jenkins-x.io/developing/create-quickstart/

### Examples

  # create a new quickstart
  jx project quickstart
  
  # creates a quickstart filtering on http based ones
  jx project quickstart -f http

### Options

```
      --branches string                The branch pattern for branches to trigger CI/CD pipelines on
      --canary                         should we use canary rollouts (progressive delivery) by default for this application. e.g. using a Canary deployment via flagger. Requires the installation of flagger and istio/gloo in your cluster
      --credentials string             The Jenkins credentials name used by the job
      --deploy-kind string             The kind of deployment to use for the project. Should be one of knative, default
      --disable-updatebot              disable updatebot-maven-plugin from attempting to fix/update the maven pom.xml
      --docker-registry-org string     The name of the docker registry organisation to use. If not specified then the Git provider organisation will be used
      --dry-run                        Performs local changes to the repo but skips the import into Jenkins X
      --external-jenkins-url string    The jenkins url that an external git provider needs to use
  -f, --filter string                  The text filter
      --framework string               The framework to filter on
      --git-api-token string           The Git API token to use for creating new Git repositories
      --git-host string                The Git server host if not using GitHub when pushing created project
      --git-provider-kind string       Kind of Git server. If not specified, kind of server will be autodetected from Git provider URL. Possible values: bitbucketcloud, bitbucketserver, gitea, gitlab, github, fakegit
      --git-provider-url string        The Git server URL to create new Git repositories inside
      --git-public                     Create new Git repositories as public
      --git-username string            The Git username to use for creating new Git repositories
  -h, --help                           help for quickstart
      --hpa                            should we enable the Horizontal Pod Autoscaler for this application.
      --import-commit-message string   Specifies the initial commit message used when importing the project
  -m, --import-mode string             The import mode to use. Should be one of Jenkinsfile, YAML
      --jenkinsfile string             The name of the Jenkinsfile to use. If not specified then 'Jenkinsfile' will be used
      --jenkinsfilerunner string       if you want to import into Jenkins X with Jenkinsfilerunner this argument lets you specify the container image to use
      --jx                             if you want to default to importing this project into Jenkins X instead of a Jenkins server if you have a mixed Jenkins X and Jenkins cluster
  -l, --language string                The language to filter on
      --machine-learning               Allow machine-learning quickstarts in results
      --name string                    Specify the Git repository name to import the project into (if it is not already in one)
      --no-import                      Disable import after the creation
      --no-pack                        Disable trying to default a Dockerfile and Helm Chart from the build pack
      --org string                     Specify the Git provider organisation to import the project into (if it is not already in one)
  -g, --organisations stringArray      The GitHub organisations to query for quickstarts
  -o, --output-dir string              Directory to output the project to. Defaults to the current directory
      --owner string                   The owner to filter on
      --pack string                    The name of the build pack to use. If none is specified it will be chosen based on matching the source code languages
  -p, --project-name string            The project name (for use with -b batch mode)
      --scheduler string               The name of the Scheduler configuration to use for ChatOps when using Prow
      --service-account string         The Kubernetes ServiceAccount to use to run the initial pipeline (default "tekton-bot")
  -t, --tag stringArray                The tags on the quickstarts to filter
      --use-default-git                use default git account
```

### SEE ALSO

* [project](project.md)	 - Create a new project by importing code, creating a quickstart or custom wizard for spring

###### Auto generated by spf13/cobra on 1-Jul-2020
