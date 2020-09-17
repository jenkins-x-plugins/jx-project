package testimports

import (
	"github.com/jenkins-x/go-scm/scm"
	fakescm "github.com/jenkins-x/go-scm/scm/driver/fake"
	fakejx "github.com/jenkins-x/jx-api/pkg/client/clientset/versioned/fake"
	fakeinput "github.com/jenkins-x/jx-helpers/pkg/input/fake"
	"github.com/jenkins-x/jx-helpers/pkg/kube/jxenv"
	"github.com/jenkins-x/jx-project/pkg/cmd/importcmd"
	"k8s.io/client-go/kubernetes/fake"
)

func SetFakeClients(o *importcmd.ImportOptions) {
	fakeInput := &fakeinput.FakeInput{
		Values: map[string]string{},
	}
	o.Input = fakeInput
	o.ScmClient = createFakeScmClient()

	ns := "jx"
	devEnv := jxenv.CreateDefaultDevEnvironment(ns)
	devEnv.Namespace = ns
	devEnv.Spec.Source.URL = "https://github.com/myorg/myrepo.git"

	o.KubeClient = fake.NewSimpleClientset()
	o.JXClient = fakejx.NewSimpleClientset(devEnv)
	o.Namespace = ns
}

func createFakeScmClient() *scm.Client {
	/*
		testOrgName := "jstrachan"
		testRepoName := "myrepo"
		stagingRepoName := "environment-staging"
		prodRepoName := "environment-production"

		fakeRepo, _ := gits.NewFakeRepository(testOrgName, testRepoName, nil, nil)
		stagingRepo, _ := gits.NewFakeRepository(testOrgName, stagingRepoName, nil, nil)
		prodRepo, _ := gits.NewFakeRepository(testOrgName, prodRepoName, nil, nil)

		fakeScmClient := gits.NewFakeProvider(fakeRepo, stagingRepo, prodRepo)
		userAuth := auth.UserAuth{
			Username:    "jx-testing-user",
			ApiToken:    "someapitoken",
			BearerToken: "somebearertoken",
			Password:    "password",
		}
		authServer := auth.AuthServer{
			Users:       []*auth.UserAuth{&userAuth},
			CurrentUser: userAuth.Username,
			URL:         "https://github.com",
			Kind:        gits.KindGitHub,
			Name:        "jx-testing-server",
		}
		fakeScmClient.Server = authServer
		return fakeScmClient

	*/

	scmClient, _ := fakescm.NewDefault()
	return scmClient
}
