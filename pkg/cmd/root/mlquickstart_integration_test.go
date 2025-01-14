//go:build integration
// +build integration

package root_test

import (
	"path/filepath"
	"testing"

	"github.com/jenkins-x-plugins/jx-project/pkg/cmd/root"
	"github.com/jenkins-x-plugins/jx-project/pkg/cmd/testimports"
	"github.com/stretchr/testify/assert"
)

func TestCreateMLQuickstartProjects(t *testing.T) {
	testDir := t.TempDir()

	appName := "mymlapp"

	_, o := root.NewCmdCreateMLQuickstart()
	o.Filter.Text = "ML-python-pytorch-cpu"
	o.Filter.ProjectName = appName

	testimports.SetFakeClients(t, &o.Options.ImportOptions, false)

	o.Dir = testDir
	o.OutDir = testDir
	o.DisableMaven = true
	o.Repository = appName
	o.WaitForSourceRepositoryPullRequest = false

	err := o.Run()
	assert.NoError(t, err)
	if err == nil {
		appName1 := appName + "-service"
		appDir1 := filepath.Join(testDir, appName1)
		assert.FileExists(t, filepath.Join(appDir1, "Dockerfile"))
		assert.FileExists(t, filepath.Join(appDir1, "charts", appName1, "Chart.yaml"))
		assert.FileExists(t, filepath.Join(appDir1, ".lighthouse", "jenkins-x", "triggers.yaml"))

		appName2 := appName + "-training"
		appDir2 := filepath.Join(testDir, appName2)
		assert.FileExists(t, filepath.Join(appDir2, "Dockerfile"))
		assert.FileExists(t, filepath.Join(appDir2, ".lighthouse", "jenkins-x", "triggers.yaml"))
	}
}
