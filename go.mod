module github.com/jenkins-x/jx-project

go 1.15

require (
	github.com/Azure/draft v0.15.0
	github.com/alexflint/go-filemutex v0.0.0-20171028004239-d358565f3c3f
	github.com/aws/aws-sdk-go v1.36.1 // indirect
	github.com/blang/semver v3.5.1+incompatible
	github.com/cenkalti/backoff v2.2.1+incompatible
	github.com/cpuguy83/go-md2man v1.0.10
	github.com/danwakefield/fnmatch v0.0.0-20160403171240-cbb64ac3d964 // indirect
	github.com/denormal/go-gitignore v0.0.0-20180713143441-75ce8f3e513c
	github.com/google/uuid v1.1.4
	github.com/jbrukh/bayesian v0.0.0-20200318221351-d726b684ca4a // indirect
	github.com/jenkins-x/go-scm v1.6.7
	github.com/jenkins-x/jx-api/v4 v4.0.25
	github.com/jenkins-x/jx-gitops v0.2.36
	github.com/jenkins-x/jx-helpers/v3 v3.0.93
	github.com/jenkins-x/jx-logging/v3 v3.0.3
	github.com/jenkins-x/jx-promote v0.0.247
	github.com/jenkins-x/lighthouse-client v0.0.85
	github.com/pkg/errors v0.9.1
	github.com/spf13/cobra v1.1.1
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.7.0
	github.com/tektoncd/pipeline v0.20.0
	gopkg.in/AlecAivazis/survey.v1 v1.8.8
	k8s.io/api v0.20.5
	k8s.io/apimachinery v0.20.5
	k8s.io/client-go v11.0.1-0.20190805182717-6502b5e7b1b5+incompatible
	k8s.io/helm v2.16.10+incompatible
	knative.dev/pkg v0.0.0-20210107022335-51c72e24c179
	sigs.k8s.io/yaml v1.2.0
)

replace (
	// override the go-scm from tekton
	github.com/jenkins-x/go-scm => github.com/jenkins-x/go-scm v1.6.7
	github.com/tektoncd/pipeline => github.com/jenkins-x/pipeline v0.3.2-0.20210118090417-1e821d85abf6
	k8s.io/api => k8s.io/api v0.20.2
	k8s.io/apimachinery => k8s.io/apimachinery v0.20.2
	k8s.io/client-go => k8s.io/client-go v0.20.2
	knative.dev/pkg => github.com/jstrachan/pkg v0.0.0-20210118084935-c7bdd6c14bd0
)
