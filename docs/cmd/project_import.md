## project import

Imports a local project or Git repository into Jenkins

### Usage

```
project import
```

### Synopsis

Imports a local folder or Git repository into Jenkins X. 

If you specify no other options or arguments then the current directory is imported. Or you can use '--dir' to specify a directory to import. 

You can specify the git URL as an argument. 

For more documentation see: https://jenkins-x.io/docs/using-jx/creating/import/

### Examples

  # Import the current folder
  jx project import
  
  # Import a different folder
  jx project import /foo/bar
  
  # Import a Git repository from a URL
  jx project import --url https://github.com/jenkins-x/spring-boot-web-example.git
  
  # Select a number of repositories from a GitHub organisation
  jx project import --github --org myname
  
  # Import all repositories from a GitHub organisation selecting ones to not import
  jx project import --github --org myname --all
  
  # Import all repositories from a GitHub organisation which contain the text foo
  jx project import --github --org myname --all --filter foo

### Options

```
      --all                            If selecting projects to import from a Git provider this defaults to selecting them all
  -b, --batch-mode                     Runs in batch mode without prompting for user input
      --branches string                The branch pattern for branches to trigger CI/CD pipelines on
      --canary                         should we use canary rollouts (progressive delivery) by default for this application. e.g. using a Canary deployment via flagger. Requires the installation of flagger and istio/gloo in your cluster
  -c, --credentials string             The Jenkins credentials name used by the job
      --deploy-kind string             The kind of deployment to use for the project. Should be one of knative, default
      --disable-updatebot              disable updatebot-maven-plugin from attempting to fix/update the maven pom.xml
      --docker-registry-org string     The name of the docker registry organisation to use. If not specified then the Git provider organisation will be used
      --dry-run                        Performs local changes to the repo but skips the import into Jenkins X
      --filter string                  If selecting projects to import from a Git provider this filters the list of repositories
      --git-kind string                the kind of git server to connect to
      --git-provider-url string        Deprecated: please use --git-server
      --git-server string              the git server URL to create the scm client
      --git-token string               the git token used to operate on the git repository. If not specified it's loaded from the git credentials file
      --git-user string                the git username used to operate on the git repository
      --github                         If you wish to pick the repositories from GitHub to import
  -h, --help                           help for import
      --hpa                            should we enable the Horizontal Pod Autoscaler for this application.
      --import-commit-message string   Specifies the initial commit message used when importing the project
  -m, --import-mode string             The import mode to use. Should be one of Jenkinsfile, YAML
      --jenkins string                 The name of the Jenkin server provisioned by the Jenkins Operator
  -j, --jenkinsfile string             The name of the Jenkinsfile to use. If not specified then 'Jenkinsfile' will be used
      --jenkinsfilerunner string       if you want to import into Jenkins X with Jenkinsfilerunner this argument lets you specify the container image to use
      --jx                             if you want to default to importing this project into Jenkins X instead of a Jenkins server if you have a mixed Jenkins X and Jenkins cluster
      --name string                    Specify the Git repository name to import the project into (if it is not already in one) (default "n")
      --name-label string              The kubernetes label used to specify the Jenkins service name (default "jenkins-cr")
      --no-dev-pr                      disables generating a Pull Request on the development git repository
      --no-pack                        Disable trying to default a Dockerfile and Helm Chart from the build pack
      --no-start                       disables starting a release pipeline when imprting/creating a new project
      --org string                     Specify the Git provider organisation to import the project into (if it is not already in one)
      --pack string                    The name of the build pack to use. If none is specified it will be chosen based on matching the source code languages
      --pr-poll-period duration        the time between polls of the Pull Request on the development environment git repository (default 20s)
      --pr-poll-timeout duration       the maximum amount of time we wait for the Pull Request on the development environment git repository (default 20m0s)
      --scheduler string               The name of the Scheduler configuration to use for ChatOps when using Prow
      --selector string                The kubernetes label selector to find the Jenkins Operator Services for Jenkins HTTP servers (default "app=jenkins-operator")
      --service-account string         The Kubernetes ServiceAccount to use to run the initial pipeline (default "tekton-bot")
  -u, --url string                     The git clone URL to clone into the current directory and then import
      --use-default-git                use default git account
      --wait-for-pr                    waits for the Pull Request generated on the development envirionment git repository to merge (default true)
```

### SEE ALSO

* [project](project.md)	 - Create a new project by importing code, creating a quickstart or custom wizard for spring

###### Auto generated by spf13/cobra on 23-Sep-2020
