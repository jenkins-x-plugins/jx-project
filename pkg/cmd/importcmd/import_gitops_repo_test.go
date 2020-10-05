// +build integration

package importcmd_test

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/jenkins-x/jx-helpers/v3/pkg/files"
	"github.com/jenkins-x/jx-helpers/v3/pkg/kube/jxenv"
	"github.com/jenkins-x/jx-helpers/v3/pkg/kube/naming"
	"github.com/jenkins-x/jx-project/pkg/cmd/importcmd"
	"github.com/jenkins-x/jx-project/pkg/cmd/testimports"

	v1 "github.com/jenkins-x/jx-api/v3/pkg/apis/jenkins.io/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestImportGitOpsRepository(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "test-import-jx-gha-")
	assert.NoError(t, err)

	name := "import_gitops_repo"
	srcDir := filepath.Join("test_data", name)
	require.DirExists(t, srcDir, "source dir does not exist")

	buildPackURL := "https://github.com/jenkins-x/jxr-packs-kubernetes.git"

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
}
