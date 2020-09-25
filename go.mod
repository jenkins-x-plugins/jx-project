module github.com/jenkins-x/jx-project

go 1.13

require (
	github.com/Azure/draft v0.15.0
	github.com/alexflint/go-filemutex v0.0.0-20171028004239-d358565f3c3f
	github.com/blang/semver v3.5.1+incompatible
	github.com/cpuguy83/go-md2man v1.0.10
	github.com/danwakefield/fnmatch v0.0.0-20160403171240-cbb64ac3d964 // indirect
	github.com/denormal/go-gitignore v0.0.0-20180713143441-75ce8f3e513c
	github.com/google/uuid v1.1.1
	github.com/jbrukh/bayesian v0.0.0-20200318221351-d726b684ca4a // indirect
	github.com/jenkins-x/go-scm v1.5.175
	github.com/jenkins-x/jx-api v0.0.23
	github.com/jenkins-x/jx-gitops v0.0.326
	github.com/jenkins-x/jx-helpers v1.0.74
	github.com/jenkins-x/jx-logging v0.0.11
	github.com/jenkins-x/jx-promote v0.0.120
	github.com/pkg/errors v0.9.1
	github.com/spf13/cobra v1.0.0
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.6.1
	github.com/tektoncd/pipeline v0.14.2
	gopkg.in/AlecAivazis/survey.v1 v1.8.8
	k8s.io/api v0.18.1
	k8s.io/apimachinery v0.18.1
	k8s.io/client-go v11.0.1-0.20190805182717-6502b5e7b1b5+incompatible
	k8s.io/helm v2.16.10+incompatible
	knative.dev/pkg v0.0.0-20200702222342-ea4d6e985ba0
	sigs.k8s.io/yaml v1.2.0

)

replace (
	k8s.io/api => k8s.io/api v0.17.6
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.17.6
	k8s.io/apimachinery => k8s.io/apimachinery v0.17.6
	k8s.io/client-go => k8s.io/client-go v0.17.6
	k8s.io/kubernetes => k8s.io/kubernetes v1.14.7
)
