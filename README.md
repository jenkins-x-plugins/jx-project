# jwizard

[![Documentation](https://godoc.org/github.com/jenkins-x-labs/jwizard?status.svg)](http://godoc.org/github.com/jenkins-x-labs/jwizard)
[![Go Report Card](https://goreportcard.com/badge/github.com/jenkins-x-labs/jwizard)](https://goreportcard.com/report/github.com/jenkins-x-labs/jwizard)

JWizard is an experimental CLI to allow quickstarts to be created and repositories to be imported into either [Jenkins](https://jenkins.io/) servers that are setup and managed via GitOps and the [Jenkins Operator](https://jenkinsci.github.io/kubernetes-operator/) or [Jenkins X](https://jenkins-x.io/).

The idea is to provide a single developer UX around creating quickstarts and importing repositories whether you use just Jenkins or just Jenkins X or a combination of both.

## Getting Started

To setup a Kubernetes cluster with Jenkins and/or Jenkins X installed please follow the  [helmboot getting started guide](https://github.com/jenkins-x-labs/helmboot#creating-a-new-installation).

In particularly make sure you use `helmboot --jenkins` if you want to try out the integration between Jenkins and Jenkins X.

## Importing Repositories and Creating Quickstarts

Just run the `jwizard` command line and follow the instructions.

If you have ever seen [Jenkins X](https://jenkins-x.io/) or have used `jx import` or `jx create quickstart` you can try run those directly via:

* `jwizard quickstart`
* `jwizard import`
 
## How it works

When importing a project `jwizard` looks for a `Jenkinfile` in the source code. 

If there is no `Jenkinsfile` then the wizard assumes you wish to proceed with a [Jenkins X Pipeline](https://jenkins-x.io/docs/concepts/jenkins-x-pipelines/) based on Tekton and imports it in the usual Jenkins X way. You also get to confirm the kind of build pack and language you wish to use for the automated CI/CD - so its easy to import any workload whether its a library, a binary, a container image, a helm chart or a fully blown microservice for automated kubernetes based CI/CD.

If a `Jenkinsfile` is present  then the wizard assumes you may wish to use a Jenkins server or [Jenkinsfile Runner](https://github.com/jenkinsci/jenkinsfile-runner) to run the pipelines, so it presents you with a list of the available Jenkins options to choose from. 

When using a Jenkins Server you get two options:

* use vanilla Jenkins pipelines via `Multi Branch Project` to perform the webhook handling and run the pipelines
* use  [lighthouse](https://github.com/jenkins-x/lighthouse) for webhook handling and ChatOps on Pull Requests. Then when a pipeline is triggered we use the [trigger-pipeline](https://github.com/jenkins-x-labs/trigger-pipeline) as a step to run the pipeline remotely inside a specific Jenkins server (without using the `Multi Branch Project`).

### Supported Integrations

When importing a project these approaches are supported:

* [Jenkins X Pipeline](https://jenkins-x.io/docs/concepts/jenkins-x-pipelines/) using Tekton 
* Jenkins pipelines via `Multi Branch Project`
* [lighthouse](https://github.com/jenkins-x/lighthouse) for ChatOps triggering a remote Jenkins pipeline via [trigger-pipeline](https://github.com/jenkins-x-labs/trigger-pipeline) (without using `Multi Branch Project`)
* [Jenkinsfile Runner](https://github.com/jenkinsci/jenkinsfile-runner) based pipelines in Tekton. You can override the container image used for the pipeline on import via the `--jenkinsfilerunner myimage:1.2.3` command line argument 
 
## Changes since `jx import`

For those of you who know [Jenkins X](https://jenkins-x.io/) and have used [jx import](https://jenkins-x.io/commands/jx_import/) before this wizard is a little different:

* the commands are a little different:
  * `jx create import` is now `jwizard import`
  * `jx create quickstart` is now `jwizard quickstart`
  * `jx create project` is now `jwizard`
  * `jx create spring` is now `jwizard spring`
* when importing to Jenkins X we ask which build pack you wish to use (e.g. classic or kubernetes) so that you can import java libraries or node modules easily in addition to kubernetes native applications
* the wizard will prompt you for the pack name (language) once the detection has occurred. Usually the pack name detection is good enough. e.g. detecting `maven` but you may wish to change the version of the pack (e.g. `maven-java11`)
* when importing a project and you are using Jenkins X and Jenkins in the same cluster you get asked whether you want to import the project into [Jenkins X](https://jenkins-x.io/) or to pick which Jenkins server to use
* we support 2 modes of importing projects to Jenkins
  * regular Jenkins import where a Multi Branch Project is used and Jenkins processes the webhooks
  * ChatOps mode: we use [lighthouse](https://github.com/jenkins-x/lighthouse) to handle the webhooks and ChatOps and then when triggered we trigger regular pipelines inside the Jenkins server 
* if your repository contains a `Jenkinsfile` and you choose to import into a Jenkins server we don't run the build packs and generate a `Dockerfile`, helm chart or `jenkins-x.yml`
