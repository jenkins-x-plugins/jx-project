//go:build integration
// +build integration

package importcmd_test

import (
	"context"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/jenkins-x-plugins/jx-project/pkg/cmd/importcmd"
	"github.com/jenkins-x-plugins/jx-project/pkg/cmd/testimports"
	"github.com/jenkins-x/go-scm/scm"
	v1 "github.com/jenkins-x/jx-api/v4/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx-helpers/v3/pkg/files"
	"github.com/jenkins-x/jx-helpers/v3/pkg/kube/jxenv"
	"github.com/jenkins-x/jx-helpers/v3/pkg/kube/naming"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestImportTektonCatalogProject(t *testing.T) {
	tempDir := t.TempDir()

	testData := path.Join("test_data", "import_projects")
	_, err := os.Stat(testData)
	assert.NoError(t, err)

	name := "nodejs"
	srcDir := filepath.Join(testData, name)
	assert.DirExists(t, srcDir, "source dir does not exist")

	testDir := tempDir

	files.CopyDir(srcDir, testDir, true)
	_, dirName := filepath.Split(testDir)
	dirName = naming.ToValidName(dirName)
	_, o := importcmd.NewCmdImportAndOptions()

	testimports.SetFakeClients(t, o, false)

	o.Dir = testDir
	o.DisableMaven = true
	o.UseDefaultGit = true
	o.WaitForSourceRepositoryPullRequest = false

	o.Destination.JenkinsX.Enabled = true
	callback := func(env *v1.Environment) error {
		return nil
	}
	err = jxenv.ModifyDevEnvironment(o.KubeClient, o.JXClient, o.Namespace, callback)
	require.NoError(t, err, "failed to modify Dev Environment")

	err = o.Run()
	require.NoError(t, err, "Failed %s with %s", dirName, err)

	// lighthouse tekton pipelines...
	assert.FileExists(t, filepath.Join(testDir, ".lighthouse", "jenkins-x", "triggers.yaml"))
	assert.FileExists(t, filepath.Join(testDir, ".lighthouse", "jenkins-x", "release.yaml"))
	assert.FileExists(t, filepath.Join(testDir, ".lighthouse", "jenkins-x", "pullrequest.yaml"))
	assert.NoFileExists(t, filepath.Join(testDir, ".lighthouse", "jenkins-x", "Kptfile"))
	assert.NoFileExists(t, filepath.Join(testDir, "jenkins-x.yml"))

	assert.FileExists(t, filepath.Join(testDir, "Dockerfile"))
	assert.FileExists(t, filepath.Join(testDir, "charts", dirName, "Chart.yaml"))
	assert.FileExists(t, filepath.Join(testDir, "charts", dirName, "templates", "deployment.yaml"))

	// let's verify the pipeline bot user is a collaborator on the repository
	require.NotNil(t, o.BootScmClient, "should have created a boot SCM client")

	ctx := context.Background()
	_, repoFullName := filepath.Split(tempDir)
	flag, _, err := o.ScmFactory.ScmClient.Repositories.IsCollaborator(ctx, repoFullName, testimports.PipelineUsername)
	require.NoError(t, err, "failed to check for collaborator for repo %s user %s", repoFullName, testimports.PipelineUsername)
	assert.True(t, flag, "should be a collaborator for repo %s user %s", repoFullName, testimports.PipelineUsername)

	envRepo := "jenkins-x-labs-bdd-tests/jx3-gke-gsm"
	prs, _, err := o.ScmFactory.ScmClient.PullRequests.List(ctx, envRepo, &scm.PullRequestListOptions{Open: true, Closed: true})
	require.NoError(t, err, "failed to find dev env repo %s", envRepo)
	require.Len(t, prs, 1, "should have found a Pull Request for dev env repo %s", envRepo)

	pr := prs[0]
	labels := pr.Labels
	require.NotEmpty(t, labels, "should labels Pull Request for dev env repo %s #%d", envRepo, pr.Number)
	var labelValues []string
	for _, label := range labels {
		labelValues = append(labelValues, label.Name)
	}
	t.Logf("Pull Request #%d for dev env repo %s has labels: %v", pr.Number, envRepo, labelValues)
	assert.Equal(t, []string{"env/dev"}, labelValues, "Pull Request labels for #%d on dev env repo %s", pr.Number, envRepo)
}
