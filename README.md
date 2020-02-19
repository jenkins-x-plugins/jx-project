# jwizard

[![Documentation](https://godoc.org/github.com/jenkins-x-labs/jwizard?status.svg)](http://godoc.org/github.com/jenkins-x-labs/jwizard)
[![Go Report Card](https://goreportcard.com/badge/github.com/jenkins-x-labs/jwizard)](https://goreportcard.com/report/github.com/jenkins-x-labs/jwizard)

JWizard is an experimental CLI to allow quickstarts to be created and repositories to be imported into either [Jenkins](https://jenkins.io/) servers that are setup and managed via GitOps and the [Jenkins Operator](https://jenkinsci.github.io/kubernetes-operator/) or [Jenkins X](https://jenkins-x.io/).

The idea is to provide a single developer UX around creating quickstarts and importing repositories whether you use just Jenkins or just Jenkins X or a combination of both.


## Changes since `jx import`

For those of you who know [Jenkins X](https://jenkins-x.io/) and have used [jx import](https://jenkins-x.io/commands/jx_import/) before this wizard is a little different:

* the commands are a little different:
  * `jx create import` is now `jwizard import`
  * `jx create quickstart` is now `jwizard quickstart`
  * `jx create project` is now `jwizard`
  * `jx create spring` is now `jwizard spring`
* when importing to Jenkins X we ask which build pack you wish to use (e.g. classic or kubernetes) so that you can import java libraries or node modules easily in addition to kubernetes native applications
* we prompt you for the pack name once the detection has occurred. Usually the pack name detection is good enough. e.g. detecting `maven` but you may wish to change the version of the pack (e.g. `maven-java11`)
* when importing a project and you are using Jenkins X and Jenkins in the same cluster you get asked whether you want to import the project into [Jenkins X](https://jenkins-x.io/) or to pick which Jenkins server to use
* we support 2 modes of importing projects to Jenkins
  * regular Jenkins import where a Multi Branch Project is used and Jenkins processes the webhooks
  * ChatOps mode: we use [lighthouse](https://github.com/jenkins-x/lighthouse) to handle the webhooks and ChatOps and then when triggered we trigger regular pipelines inside the Jenkins server 
* if your repository contains a `Jenkinsfile` and you choose to import into a Jenkins server we don't run the build packs and generate a `Dockerfile`, helm chart or `jenkins-x.yml`



## Getting Started

Start off creating a Kubernetes Cluster 