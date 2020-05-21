// +build integration

package importcmd_test

import (
	"github.com/jenkins-x-labs/jwizard/pkg/cmd/fakejxfactory"
	"github.com/jenkins-x-labs/jwizard/pkg/cmd/importcmd"
	"github.com/jenkins-x/jx/pkg/cmd/testhelpers"
	"github.com/jenkins-x/jx/pkg/config"
	"github.com/jenkins-x/jx/pkg/kube/naming"

	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"

	v1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	fake_clients "github.com/jenkins-x/jx/pkg/cmd/clients/fake"
	"github.com/jenkins-x/jx/pkg/cmd/opts"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/helm"
	resources_test "github.com/jenkins-x/jx/pkg/kube/resources/mocks"
	"github.com/jenkins-x/jx/pkg/tests"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/stretchr/testify/assert"
)

func TestImportGitHubActionProject(t *testing.T) {
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

	tempDir, err := ioutil.TempDir("", "test-import-jx-gha-")
	assert.NoError(t, err)

	testData := path.Join("test_data", "import_projects")
	_, err = os.Stat(testData)
	assert.NoError(t, err)

	name := "nodejs"
	srcDir := filepath.Join(testData, name)
	assert.DirExists(t, srcDir, "source dir does not exist")

	buildPackURL := "https://github.com/jstrachan/fake-github-action-build-pack.git"

	testDir := tempDir

	util.CopyDir(srcDir, testDir, true)
	_, dirName := filepath.Split(testDir)
	dirName = naming.ToValidName(dirName)
	o := &importcmd.ImportOptions{
		CommonOptions: &opts.CommonOptions{},
	}

	o.SetFactory(fake_clients.NewFakeFactory())
	o.JXFactory = fakejxfactory.NewFakeFactory()
	o.GitProvider = createFakeGitProvider()

	k8sObjects := []runtime.Object{}
	jxObjects := []runtime.Object{}
	helmer := helm.NewHelmCLI("helm", helm.V3, dirName, true)
	testhelpers.ConfigureTestOptionsWithResources(o.CommonOptions, k8sObjects, jxObjects, gits.NewGitCLI(), nil, helmer, resources_test.NewMockInstaller())
	if o.Out == nil {
		o.Out = tests.Output()
	}
	if o.Out == nil {
		o.Out = os.Stdout
	}
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
	err = o.ModifyDevEnvironment(callback)
	require.NoError(t, err, "failed to modify Dev Environment")

	err = o.Run()
	require.NoError(t, err, "Failed %s with %s", dirName, err)

	assert.FileExists(t, filepath.Join(testDir, "Dockerfile"))
	assert.FileExists(t, filepath.Join(testDir, "charts", dirName, "Chart.yaml"))
	assert.FileExists(t, filepath.Join(testDir, ".github", "pullrequest", "task.yml"))
	assert.FileExists(t, filepath.Join(testDir, ".github", "release", "task.yml"))
	assert.FileExists(t, filepath.Join(testDir, config.ProjectConfigFileName))

	projectConfig, projectFileName, err := config.LoadProjectConfig(testDir)
	require.NoError(t, err, "could not load jenkins configuration at %s", testDir)

	assert.Equal(t, "none", projectConfig.BuildPack, "buildPack property in file %s", projectFileName)
}
