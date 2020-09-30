// +build integration

package importcmd_test

import (
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"testing"

	v1 "github.com/jenkins-x/jx-api/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx-helpers/pkg/files"
	"github.com/jenkins-x/jx-helpers/pkg/kube/jxenv"
	"github.com/jenkins-x/jx-helpers/pkg/kube/naming"
	"github.com/jenkins-x/jx-project/pkg/cmd/importcmd"
	"github.com/jenkins-x/jx-project/pkg/cmd/testimports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestImportTektonCatalogProject(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "test-import-jx-gha-")
	assert.NoError(t, err)

	testData := path.Join("test_data", "import_projects")
	_, err = os.Stat(testData)
	assert.NoError(t, err)

	name := "nodejs"
	srcDir := filepath.Join(testData, name)
	assert.DirExists(t, srcDir, "source dir does not exist")

	buildPackURL := "https://github.com/jenkins-x/jx3-pipeline-catalog.git"

	testDir := tempDir

	files.CopyDir(srcDir, testDir, true)
	_, dirName := filepath.Split(testDir)
	dirName = naming.ToValidName(dirName)
	o := &importcmd.ImportOptions{}

	testimports.SetFakeClients(t, o)

	o.Dir = testDir
	o.DisableMaven = true
	o.UseDefaultGit = true

	o.Destination.JenkinsX.Enabled = true
	callback := func(env *v1.Environment) error {
		env.Spec.TeamSettings.ImportMode = v1.ImportModeTypeYAML
		if buildPackURL != "" {
			env.Spec.TeamSettings.BuildPackURL = buildPackURL
		}
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
	assert.NoFileExists(t, filepath.Join(testDir, "jenkins-x.yml"))

	// TODO
	//assert.FileExists(t, filepath.Join(testDir, "Dockerfile"))
	assert.FileExists(t, filepath.Join(testDir, "charts", dirName, "Chart.yaml"))
	assert.FileExists(t, filepath.Join(testDir, "charts", dirName, "templates", "deployment.yaml"))
}
