// +build integration

package importcmd_test

import (
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/jenkins-x/jx-helpers/pkg/files"
	"github.com/jenkins-x/jx-helpers/pkg/kube/jxenv"
	"github.com/jenkins-x/jx-helpers/pkg/kube/naming"
	"github.com/jenkins-x/jx-project/pkg/cmd/importcmd"
	"github.com/jenkins-x/jx-project/pkg/cmd/testimports"
	"github.com/jenkins-x/jx-project/pkg/config"

	v1 "github.com/jenkins-x/jx-api/pkg/apis/jenkins.io/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestImportGoLangProject(t *testing.T) {
	/* TODO
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

	tempDir, err := ioutil.TempDir("", "test-import-jx-gha-")
	assert.NoError(t, err)

	testData := path.Join("test_data", "import_projects")
	_, err = os.Stat(testData)
	assert.NoError(t, err)

	name := "golang"
	srcDir := filepath.Join(testData, name)
	assert.DirExists(t, srcDir, "source dir does not exist")

	buildPackURL := "https://github.com/jenkins-x/jxr-packs-kubernetes.git"

	testDir := tempDir

	files.CopyDir(srcDir, testDir, true)
	_, dirName := filepath.Split(testDir)
	dirName = naming.ToValidName(dirName)
	o := &importcmd.ImportOptions{}

	testimports.SetFakeClients(o)
	o.Dir = testDir
	o.DryRun = true
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

	assert.FileExists(t, filepath.Join(testDir, "Dockerfile"))
	assert.FileExists(t, filepath.Join(testDir, "charts", dirName, "Chart.yaml"))
	assert.FileExists(t, filepath.Join(testDir, "charts", dirName, "resources", "README.md"))
	assert.FileExists(t, filepath.Join(testDir, "preview", "helmfile.yaml"))
	assert.FileExists(t, filepath.Join(testDir, config.ProjectConfigFileName))

	projectConfig, projectFileName, err := config.LoadProjectConfig(testDir)
	require.NoError(t, err, "could not load jenkins configuration at %s", testDir)

	assert.Equal(t, "go", projectConfig.BuildPack, "buildPack property in file %s", projectFileName)
}
