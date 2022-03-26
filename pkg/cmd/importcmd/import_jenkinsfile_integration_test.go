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
	v1 "github.com/jenkins-x/jx-api/v4/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx-helpers/v3/pkg/files"
	"github.com/jenkins-x/jx-helpers/v3/pkg/kube/jxenv"
	"github.com/jenkins-x/jx-helpers/v3/pkg/kube/naming"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestImportJenkinsfileProject(t *testing.T) {
	tempDir := t.TempDir()

	testData := path.Join("test_data", "import_projects")
	_, err := os.Stat(testData)
	assert.NoError(t, err)

	name := "custom_jenkins"
	srcDir := filepath.Join(testData, name)
	assert.DirExists(t, srcDir, "source dir does not exist")

	testDir := tempDir

	files.CopyDir(srcDir, testDir, true)
	_, dirName := filepath.Split(testDir)
	dirName = naming.ToValidName(dirName)
	_, o := importcmd.NewCmdImportAndOptions()

	testimports.SetFakeClients(t, o, false)

	o.BatchMode = true
	o.Dir = testDir
	o.DisableMaven = true
	o.UseDefaultGit = true
	o.WaitForSourceRepositoryPullRequest = false

	callback := func(env *v1.Environment) error {
		env.Spec.Source.URL = "https://fake.git/jenkins-x-labs-bdd-tests/jx3-kubernetes-jenkins"
		return nil
	}
	err = jxenv.ModifyDevEnvironment(o.KubeClient, o.JXClient, o.Namespace, callback)
	require.NoError(t, err, "failed to modify Dev Environment")

	o.Destination.Jenkins.Enabled = true
	err = o.Run()
	require.NoError(t, err, "Failed %s with %s", dirName, err)

	// lighthouse tekton pipelines...
	//assert.FileExists(t, filepath.Join(testDir, ".lighthouse", "jenkins-x", "triggers.yaml"))

	// lets verify the pipeline bot user is a collaborator on the repository
	require.NotNil(t, o.BootScmClient, "should have created a boot SCM client")

	ctx := context.Background()
	_, repoFullName := filepath.Split(tempDir)
	flag, _, err := o.ScmFactory.ScmClient.Repositories.IsCollaborator(ctx, repoFullName, testimports.PipelineUsername)
	require.NoError(t, err, "failed to check for collaborator for repo %s user %s", repoFullName, testimports.PipelineUsername)
	assert.True(t, flag, "should be a collaborator for repo %s user %s", repoFullName, testimports.PipelineUsername)
}
