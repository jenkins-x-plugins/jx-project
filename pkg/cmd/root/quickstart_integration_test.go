// +build integration

package root_test

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/jenkins-x/jx-project/pkg/cmd/importcmd"
	"github.com/jenkins-x/jx-project/pkg/cmd/root"
	"github.com/jenkins-x/jx-project/pkg/cmd/testimports"
	"github.com/jenkins-x/jx-project/pkg/quickstarts"
	"github.com/stretchr/testify/assert"
)

func TestCreateQuickstartProjects(t *testing.T) {
	// TODO lets skip this test for now as it often fails with rate limits
	t.SkipNow()

	/**
	TODO
	originalJxHome, tempJxHome, err := testhelpers.CreateTestJxHomeDir()
	assert.NoError(t, err)
	defer func() {
		err := testhelpers.CleanupTestJxHomeDir(originalJxHome, tempJxHome)
		assert.NoError(t, err)
	}()
	originalKubeCfg, tempKubeCfg, err := testhelpers.CreateTestKubeConfigDir()
	assert.NoError(t, err)
	defer func() {
		err := testhelpers.CleanupTestKubeConfigDir(originalKubeCfg, tempKubeCfg)
		assert.NoError(t, err)
	}()
	*/

	testDir, err := ioutil.TempDir("", "test-create-quickstart")
	assert.NoError(t, err)

	appName := "myvets"

	o := &root.CreateQuickstartOptions{
		Options: root.Options{
			ImportOptions: importcmd.ImportOptions{},
		},
		GitHubOrganisations: []string{"petclinic-gcp"},
		Filter: quickstarts.QuickstartFilter{
			Text:        "petclinic-gcp/spring-petclinic-vets-service",
			ProjectName: appName,
		},
	}
	testimports.SetFakeClients(&o.Options.ImportOptions)

	o.Dir = testDir
	o.OutDir = testDir
	o.DryRun = true
	o.DisableMaven = true
	o.IgnoreTeam = true
	o.Repository = appName

	err = o.Run()
	assert.NoError(t, err)
	if err == nil {
		appDir := filepath.Join(testDir, appName)
		jenkinsfile := filepath.Join(appDir, "Jenkinsfile")
		assert.FileExists(t, jenkinsfile)
		assert.FileExists(t, filepath.Join(appDir, "Dockerfile"))
		assert.FileExists(t, filepath.Join(appDir, "charts", appName, "Chart.yaml"))
		assert.FileExists(t, filepath.Join(appDir, "charts", appName, "Makefile"))
		assert.NoFileExists(t, filepath.Join(appDir, "charts", "spring-petclinic-vets-service", "Chart.yaml"))
	}
}
