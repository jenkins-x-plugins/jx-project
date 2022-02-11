module github.com/jenkins-x-plugins/jx-project

go 1.15

require (
	github.com/Azure/draft v0.15.0
	github.com/alexflint/go-filemutex v0.0.0-20171028004239-d358565f3c3f
	github.com/blang/semver v3.5.1+incompatible
	github.com/cenkalti/backoff v2.2.1+incompatible
	github.com/cpuguy83/go-md2man v1.0.10
	github.com/danwakefield/fnmatch v0.0.0-20160403171240-cbb64ac3d964 // indirect
	github.com/denormal/go-gitignore v0.0.0-20180713143441-75ce8f3e513c
	github.com/google/uuid v1.2.0
	github.com/jbrukh/bayesian v0.0.0-20200318221351-d726b684ca4a // indirect
	github.com/jenkins-x-plugins/jx-gitops v0.4.2
	github.com/jenkins-x-plugins/jx-promote v0.0.278
	github.com/jenkins-x/go-scm v1.11.2
	github.com/jenkins-x/jx-api/v4 v4.3.0
	github.com/jenkins-x/jx-helpers/v3 v3.1.1
	github.com/jenkins-x/jx-logging/v3 v3.0.6
	github.com/jenkins-x/lighthouse-client v0.0.295
	github.com/pkg/errors v0.9.1
	github.com/spf13/cobra v1.2.0
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.7.0
	github.com/tektoncd/pipeline v0.26.0
	gopkg.in/AlecAivazis/survey.v1 v1.8.8
	helm.sh/helm/v3 v3.6.3
	k8s.io/api v0.21.0
	k8s.io/apimachinery v0.21.0
	k8s.io/client-go v11.0.1-0.20190805182717-6502b5e7b1b5+incompatible
	knative.dev/pkg v0.0.0-20210730172132-bb4aaf09c430
	sigs.k8s.io/yaml v1.2.0
)

replace (
	// helm dependencies
	github.com/docker/distribution => github.com/docker/distribution v0.0.0-20191216044856-a8371794149d
	github.com/docker/docker => github.com/moby/moby v17.12.0-ce-rc1.0.20200618181300-9dc6525e6118+incompatible
	// override the go-scm from tekton
	github.com/jenkins-x/go-scm => github.com/jenkins-x/go-scm v1.11.2
	github.com/jenkins-x/jx-helpers/v3 => github.com/jenkins-x/jx-helpers/v3 v3.1.1
	// for the PipelineRun debug fix see: https://github.com/tektoncd/pipeline/pull/4145
	github.com/tektoncd/pipeline => github.com/jstrachan/pipeline v0.21.1-0.20210811150720-45a86a5488af
	k8s.io/api => k8s.io/api v0.20.7
	k8s.io/apimachinery => k8s.io/apimachinery v0.20.7
	k8s.io/client-go => k8s.io/client-go v0.20.7
)
