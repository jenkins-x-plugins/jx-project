// +build integration

package importcmd_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/jenkins-x/jx-helpers/v3/pkg/stringhelpers"

	"github.com/jenkins-x/jx-helpers/v3/pkg/files"
	"github.com/jenkins-x/jx-helpers/v3/pkg/kube/naming"
	"github.com/jenkins-x/jx-project/pkg/cmd/importcmd"
	"github.com/jenkins-x/jx-project/pkg/cmd/testimports"
	"github.com/jenkins-x/jx-project/pkg/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestImportOldProject(t *testing.T) {
	// this will only work if using k8s connectivity as we create a separate
	// 'jx pipeline convert' command
	useRealJXConvert := false

	tempDir, err := ioutil.TempDir("", "test-import-jx-gha-")
	assert.NoError(t, err)

	testData := path.Join("test_data", "import_projects")
	_, err = os.Stat(testData)
	assert.NoError(t, err)

	name := "maven_custom_build_pack"
	dirName := naming.ToValidName(name)

	srcDir := filepath.Join(testData, name)
	assert.DirExists(t, srcDir, "source dir does not exist")

	testDir := tempDir

	files.CopyDir(srcDir, testDir, true)
	//_, dirName := filepath.Split(testDir)
	_, o := importcmd.NewCmdImportAndOptions()

	testimports.SetFakeClients(t, o, useRealJXConvert)

	// lets setup git
	g := o.Git()
	_, err = g.Command(tempDir, "init")
	require.NoError(t, err, "failed to git init dir %s", tempDir)

	gitURL := fmt.Sprintf("https://github.com/myowner/%s.git", dirName)
	_, err = g.Command(tempDir, "remote", "add", "origin", gitURL)
	require.NoError(t, err, "failed to setup git remote URL %s", gitURL)

	// fake discovering the git url
	o.DiscoveredGitURL = stringhelpers.UrlJoin(o.ScmFactory.GitServerURL, "myowner", "myrepo")
	o.BatchMode = true
	o.Dir = testDir
	o.DisableMaven = true
	o.UseDefaultGit = true
	o.WaitForSourceRepositoryPullRequest = false
	o.Destination.JenkinsX.Enabled = true

	err = o.Run()
	require.NoError(t, err, "Failed %s with %s", dirName, err)

	assert.FileExists(t, filepath.Join(testDir, "Dockerfile"))
	assert.FileExists(t, filepath.Join(testDir, "charts", dirName, "Chart.yaml"))
	//assert.FileExists(t, filepath.Join(testDir, "preview", "helmfile.yaml"))

	if useRealJXConvert {
		assert.NoFileExists(t, filepath.Join(testDir, config.ProjectConfigFileName))
		assert.FileExists(t, filepath.Join(testDir, ".lighthouse", "jenkins-x", "triggers.yaml"))
	}
}
