// +build integration

package root_test

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/jenkins-x-plugins/jx-project/pkg/cmd/root"
	"github.com/jenkins-x-plugins/jx-project/pkg/cmd/testimports"
	"github.com/stretchr/testify/assert"
)

func TestCreateQuickstartProjects(t *testing.T) {
	testDir, err := ioutil.TempDir("", "test-create-quickstart")
	assert.NoError(t, err)

	appName := "mynode"

	_, o := root.NewCmdCreateQuickstart()
	o.Filter.Text = "node-http"
	o.Filter.ProjectName = appName

	testimports.SetFakeClients(t, &o.Options.ImportOptions, false)

	o.Dir = testDir
	o.OutDir = testDir
	o.DisableMaven = true
	o.IgnoreTeam = true
	o.Repository = appName
	o.WaitForSourceRepositoryPullRequest = false

	err = o.Run()
	assert.NoError(t, err)
	if err == nil {
		appDir := filepath.Join(testDir, appName)
		pipelineFile := filepath.Join(appDir, "jenkins-x.yml")
		assert.FileExists(t, filepath.Join(appDir, "Dockerfile"))
		assert.FileExists(t, filepath.Join(appDir, "charts", appName, "Chart.yaml"))
		assert.NoFileExists(t, pipelineFile)
		assert.FileExists(t, filepath.Join(appDir, ".lighthouse", "jenkins-x", "triggers.yaml"))
	}
}
