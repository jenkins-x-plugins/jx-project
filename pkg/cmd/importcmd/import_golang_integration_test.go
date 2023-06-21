//go:build integration
// +build integration

package importcmd_test

import (
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
	"github.com/jenkins-x/jx-helpers/v3/pkg/testhelpers"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestImportGoLangProject(t *testing.T) {
	tempDir := t.TempDir()

	testData := path.Join("test_data", "import_projects")
	_, err := os.Stat(testData)
	assert.NoError(t, err)

	name := "golang"
	srcDir := filepath.Join(testData, name)
	assert.DirExists(t, srcDir, "source dir does not exist")

	testDir := tempDir

	files.CopyDir(srcDir, testDir, true)
	_, dirName := filepath.Split(testDir)
	dirName = naming.ToValidName(dirName)
	_, o := importcmd.NewCmdImportAndOptions()

	_, _, runner := testimports.SetFakeClients(t, o, false)
	o.Dir = testDir
	o.DisableMaven = true
	o.UseDefaultGit = true
	o.WaitForSourceRepositoryPullRequest = false
	o.Destination.JenkinsX.Enabled = true

	err = o.Run()
	require.NoError(t, err, "Failed %s with %s", dirName, err)

	assert.FileExists(t, filepath.Join(testDir, "Dockerfile"))
	assert.FileExists(t, filepath.Join(testDir, "charts", dirName, "Chart.yaml"))
	assert.FileExists(t, filepath.Join(testDir, "preview", "helmfile.yaml"))
	assert.NoFileExists(t, filepath.Join(testDir, config.ProjectConfigFileName))
	assert.FileExists(t, filepath.Join(testDir, ".lighthouse", "jenkins-x", "triggers.yaml"))

	// let's verify we have comments still in the values yaml
	valuesFile := filepath.Join(testDir, "charts", dirName, "values.yaml")
	assert.FileExists(t, valuesFile)
	valuesYaml, _ := testhelpers.AssertLoadFileText(t, valuesFile)
	assert.True(t, strings.Contains(valuesYaml, "# "), "values yaml should contain a comment but was: %s", valuesYaml)

	var commands []string
	found := false
	for _, c := range runner.OrderedCommands {
		cli := c.CLI()
		commands = append(commands, cli)
		if strings.HasPrefix(cli, "jx pipeline wait ") {
			found = true
		}
	}
	assert.True(t, found, "should have found a command but got %v", commands)
}
