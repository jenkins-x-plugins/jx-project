package testimports

import (
	"strings"
	"testing"

	fakescm "github.com/jenkins-x/go-scm/scm/driver/fake"
	fakejx "github.com/jenkins-x/jx-api/v3/pkg/client/clientset/versioned/fake"
	"github.com/jenkins-x/jx-helpers/v3/pkg/boot"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cmdrunner"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cmdrunner/fakerunner"
	fakeinput "github.com/jenkins-x/jx-helpers/v3/pkg/input/fake"
	"github.com/jenkins-x/jx-helpers/v3/pkg/kube/jxenv"
	"github.com/jenkins-x/jx-project/pkg/cmd/importcmd"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

const (
	// PipelineUsername the pipeline bot user name used in tests
	PipelineUsername = "my-pipeline-bot-user"
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
	devEnv.Spec.Source.URL = "https://fake.git/jx3-gitops-repositories/jx3-gke-gsm"
	devEnv.Spec.TeamSettings.PipelineUsername = PipelineUsername

	o.KubeClient = fake.NewSimpleClientset(
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      boot.SecretName,
				Namespace: "jx-git-operator",
			},
			Data: map[string][]byte{
				"url":      []byte("https://github.com/myorg/myclusterrepo"),
				"username": []byte(PipelineUsername),
				"password": []byte("dummy-pipeline-user-token"),
			},
		},
	)
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
		if c.Name == "git" && len(c.Args) > 0 {
			switch c.Args[0] {
			case "clone":
				// if we are cloning a fake URL lets switch to github
				gitURL := c.Args[1]
				if strings.HasPrefix(gitURL, "https://fake.git") {
					c.Args[1] = "https://github.com" + strings.TrimPrefix(gitURL, "https://fake.git")
				} else {
					dummyServer := "@fake.git"
					idx := strings.Index(gitURL, dummyServer)
					if idx > 0 {
						c.Args[1] = "https://github.com" + gitURL[idx+len(dummyServer):]
					}
				}
			case "push":
				// lets fake out git push
				t.Logf("faking command: %s\n", c.CLI())
				return "", nil
			}
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
