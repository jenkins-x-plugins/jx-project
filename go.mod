module github.com/jenkins-x/jx-project

go 1.15

require (
	github.com/Azure/draft v0.15.0
	github.com/alexflint/go-filemutex v0.0.0-20171028004239-d358565f3c3f
	github.com/blang/semver v3.5.1+incompatible
	github.com/cpuguy83/go-md2man v1.0.10
	github.com/danwakefield/fnmatch v0.0.0-20160403171240-cbb64ac3d964 // indirect
	github.com/denormal/go-gitignore v0.0.0-20180713143441-75ce8f3e513c
	github.com/google/uuid v1.1.1
	github.com/jbrukh/bayesian v0.0.0-20200318221351-d726b684ca4a // indirect
	github.com/jenkins-x/go-scm v1.5.190
	github.com/jenkins-x/jx-api/v3 v3.0.1
	github.com/jenkins-x/jx-gitops v0.0.409
	github.com/jenkins-x/jx-helpers/v3 v3.0.15
	github.com/jenkins-x/jx-logging/v3 v3.0.2
	github.com/jenkins-x/jx-promote v0.0.139
	github.com/pkg/errors v0.9.1
	github.com/spf13/cobra v1.0.0
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.6.1
	github.com/tektoncd/pipeline v0.14.2
	gopkg.in/AlecAivazis/survey.v1 v1.8.8
	k8s.io/api v0.19.2
	k8s.io/apimachinery v0.19.2
	k8s.io/client-go v11.0.1-0.20190805182717-6502b5e7b1b5+incompatible
	k8s.io/helm v2.16.10+incompatible
	knative.dev/pkg v0.0.0-20201002052829-735a38c03260
	sigs.k8s.io/yaml v1.2.0

)

replace (
	github.com/tektoncd/pipeline => github.com/jenkins-x/pipeline v0.3.2-0.20201002150609-ca0741e5d19a
	k8s.io/client-go => k8s.io/client-go v0.19.2
)
