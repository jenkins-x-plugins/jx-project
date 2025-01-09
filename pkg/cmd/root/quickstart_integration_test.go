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

func TestCreateQuickstartProjects(t *testing.T) {
	testDir := t.TempDir()

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

	err := o.Run()
	assert.NoError(t, err)
	if err == nil {
		appDir := filepath.Join(testDir, appName)
		assert.FileExists(t, filepath.Join(appDir, "Dockerfile"))
		assert.FileExists(t, filepath.Join(appDir, "charts", appName, "Chart.yaml"))
		assert.FileExists(t, filepath.Join(appDir, ".lighthouse", "jenkins-x", "triggers.yaml"))
	}
}

func TestCreateQuickstartProjectWithChart(t *testing.T) {
	testDir := t.TempDir()

	appName := "mynodedb"

	_, o := root.NewCmdCreateQuickstart()
	o.Filter.Text = "node-postgresql"
	o.Filter.ProjectName = appName

	testimports.SetFakeClients(t, &o.Options.ImportOptions, false)

	o.Dir = testDir
	o.OutDir = testDir
	o.DisableMaven = true
	o.IgnoreTeam = true
	o.Repository = appName
	o.WaitForSourceRepositoryPullRequest = false

	err := o.Run()
	assert.NoError(t, err)
	if err == nil {
		appDir := filepath.Join(testDir, appName)
		assert.FileExists(t, filepath.Join(appDir, "Dockerfile"))
		assert.FileExists(t, filepath.Join(appDir, "charts", appName, "Chart.yaml"))
		assert.NoFileExists(t, filepath.Join(appDir, "charts", "Chart.yaml"))
		assert.FileExists(t, filepath.Join(appDir, ".lighthouse", "jenkins-x", "triggers.yaml"))
	}
}
