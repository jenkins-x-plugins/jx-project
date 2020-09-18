package testimports

import (
	"testing"

	fakescm "github.com/jenkins-x/go-scm/scm/driver/fake"
	fakejx "github.com/jenkins-x/jx-api/pkg/client/clientset/versioned/fake"
	"github.com/jenkins-x/jx-helpers/pkg/cmdrunner"
	"github.com/jenkins-x/jx-helpers/pkg/cmdrunner/fakerunner"
	fakeinput "github.com/jenkins-x/jx-helpers/pkg/input/fake"
	"github.com/jenkins-x/jx-helpers/pkg/kube/jxenv"
	"github.com/jenkins-x/jx-project/pkg/cmd/importcmd"
	"k8s.io/client-go/kubernetes/fake"
)

// SetFakeClients sets the fake clients on the options
func SetFakeClients(t *testing.T, o *importcmd.ImportOptions) *fakescm.Data {
	fakeInput := &fakeinput.FakeInput{
		Values: map[string]string{},
	}
	o.Input = fakeInput
	client, fakeScmData := fakescm.NewDefault()
	o.ScmFactory.ScmClient = client

	// lets add a dummy token so we can create authenticated git URLs
	o.ScmFactory.GitToken = "my.fake.token"

	ns := "jx"
	devEnv := jxenv.CreateDefaultDevEnvironment(ns)
	devEnv.Namespace = ns
	devEnv.Spec.Source.URL = "https://github.com/jx3-gitops-repositories/jx3-gke-terraform-vault"

	o.KubeClient = fake.NewSimpleClientset()
	o.JXClient = fakejx.NewSimpleClientset(devEnv)
	o.Namespace = ns

	runner := NewFakeRunnerWithoutGitPush(t)
	o.CommandRunner = runner.Run
	return fakeScmData
}

// NewFakeRunnerWithoutGitPush create a fake command runner that fakes out git push
func NewFakeRunnerWithoutGitPush(t *testing.T) *fakerunner.FakeRunner {
	runner := &fakerunner.FakeRunner{}
	runner.CommandRunner = func(c *cmdrunner.Command) (string, error) {
		if c.Name == "git" && len(c.Args) > 0 && c.Args[0] == "push" {
			// lets fake out git push
			t.Logf("faking command: %s\n", c.CLI())
			return "", nil
		}

		if c.Name == "jx" && len(c.Args) > 0 && c.Args[0] == "pipeline" {
			// lets fake out starting pipelines
			t.Logf("faking command: %s\n", c.CLI())
			return "", nil
		}

		// otherwise lets do it for real
		return cmdrunner.DefaultCommandRunner(c)
	}
	return runner
}
