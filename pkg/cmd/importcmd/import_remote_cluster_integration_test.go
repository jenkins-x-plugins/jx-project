// +build integration

package importcmd_test

import (
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jenkins-x-plugins/jx-project/pkg/cmd/importcmd"
	"github.com/jenkins-x-plugins/jx-project/pkg/cmd/testimports"
	"github.com/jenkins-x-plugins/jx-project/pkg/config"
	"github.com/jenkins-x/jx-helpers/v3/pkg/files"
	"github.com/jenkins-x/jx-helpers/v3/pkg/kube/naming"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestImportRemoteCluster(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "test-import-jx-remote-")
	assert.NoError(t, err)

	srcDir := path.Join("test_data", "remote-cluster")
	require.DirExists(t, srcDir)

	testDir := tempDir
	files.CopyDir(srcDir, testDir, true)
	_, dirName := filepath.Split(testDir)
	dirName = naming.ToValidName(dirName)
	_, o := importcmd.NewCmdImportAndOptions()

	_, _, runner := testimports.SetFakeClients(t, o, false)
	o.RepoURL = "https://github.com/jx3-gitops-repositories/jx3-kubernetes-production"
	o.Dir = testDir
	o.DisableMaven = true
	o.UseDefaultGit = true
	o.WaitForSourceRepositoryPullRequest = false
	o.Destination.JenkinsX.Enabled = true

	err = o.Run()
	require.NoError(t, err, "Failed %s with %s", dirName, err)

	assert.NoFileExists(t, filepath.Join(testDir, "Dockerfile"))
	assert.NoFileExists(t, filepath.Join(testDir, "charts", dirName, "Chart.yaml"))
	assert.NoFileExists(t, filepath.Join(testDir, "preview", "helmfile.yaml"))
	assert.NoFileExists(t, filepath.Join(testDir, config.ProjectConfigFileName))
	assert.FileExists(t, filepath.Join(testDir, ".lighthouse", "jenkins-x", "triggers.yaml"))

	for _, c := range runner.OrderedCommands {
		cli := c.CLI()
		found := strings.HasPrefix(cli, "jx pipeline wait ")
		assert.False(t, found, "should not have command %s for remote repository")

		//t.Logf("got command %s\n", c.CLI())
	}
}
