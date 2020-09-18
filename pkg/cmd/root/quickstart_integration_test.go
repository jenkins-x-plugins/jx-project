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
	testDir, err := ioutil.TempDir("", "test-create-quickstart")
	assert.NoError(t, err)

	appName := "mynode"

	o := &root.CreateQuickstartOptions{
		Options: root.Options{
			ImportOptions: importcmd.ImportOptions{},
		},
		// TODO
		//GitHubOrganisations: []string{"petclinic-gcp"},
		Filter: quickstarts.QuickstartFilter{
			Text:        "node-http",
			ProjectName: appName,
		},
	}
	testimports.SetFakeClients(t, &o.Options.ImportOptions)

	o.Dir = testDir
	o.OutDir = testDir
	o.DisableMaven = true
	o.IgnoreTeam = true
	o.Repository = appName

	err = o.Run()
	assert.NoError(t, err)
	if err == nil {
		appDir := filepath.Join(testDir, appName)
		pipelineFile := filepath.Join(appDir, "jenkins-x.yml")
		assert.FileExists(t, filepath.Join(appDir, "Dockerfile"))
		assert.FileExists(t, filepath.Join(appDir, "charts", appName, "Chart.yaml"))
		assert.FileExists(t, pipelineFile)
	}
}
