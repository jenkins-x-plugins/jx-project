module github.com/jenkins-x/jx-project

go 1.15

require (
	cloud.google.com/go v0.76.0 // indirect
	github.com/Azure/draft v0.15.0
	github.com/alexflint/go-filemutex v0.0.0-20171028004239-d358565f3c3f
	github.com/aws/aws-sdk-go v1.36.1 // indirect
	github.com/blang/semver v3.5.1+incompatible
	github.com/cpuguy83/go-md2man v1.0.10
	github.com/danwakefield/fnmatch v0.0.0-20160403171240-cbb64ac3d964 // indirect
	github.com/denormal/go-gitignore v0.0.0-20180713143441-75ce8f3e513c
	github.com/google/uuid v1.1.4
	github.com/jbrukh/bayesian v0.0.0-20200318221351-d726b684ca4a // indirect
	github.com/jenkins-x/go-scm v1.5.216
	github.com/jenkins-x/jx-api/v4 v4.0.24
	github.com/jenkins-x/jx-gitops v0.1.1
	github.com/jenkins-x/jx-helpers/v3 v3.0.74
	github.com/jenkins-x/jx-logging/v3 v3.0.3
	github.com/jenkins-x/jx-promote v0.0.233
	github.com/jenkins-x/lighthouse-client v0.0.24
	github.com/pkg/errors v0.9.1
	github.com/spf13/cobra v1.1.1
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.7.0
	github.com/tektoncd/pipeline v0.20.1
	golang.org/x/crypto v0.0.0-20201221181555-eec23a3978ad // indirect
	golang.org/x/oauth2 v0.0.0-20210201163806-010130855d6c // indirect
	golang.org/x/term v0.0.0-20201210144234-2321bbc49cbf // indirect
	golang.org/x/time v0.0.0-20201208040808-7e3f01d25324 // indirect
	gopkg.in/AlecAivazis/survey.v1 v1.8.8
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b // indirect
	k8s.io/api v0.20.2
	k8s.io/apimachinery v0.20.2
	k8s.io/client-go v11.0.1-0.20190805182717-6502b5e7b1b5+incompatible
	k8s.io/helm v2.16.10+incompatible
	k8s.io/klog/v2 v2.5.0 // indirect
	k8s.io/utils v0.0.0-20210111153108-fddb29f9d009 // indirect
	knative.dev/pkg v0.0.0-20210107022335-51c72e24c179
	sigs.k8s.io/yaml v1.2.0
)

replace (
	github.com/tektoncd/pipeline => github.com/jenkins-x/pipeline v0.3.2-0.20210118090417-1e821d85abf6
	k8s.io/api => k8s.io/api v0.20.2
	k8s.io/apimachinery => k8s.io/apimachinery v0.20.2
	k8s.io/client-go => k8s.io/client-go v0.20.2
	knative.dev/pkg => github.com/jstrachan/pkg v0.0.0-20210118084935-c7bdd6c14bd0
)
