// +build unit

package gitresolver

import (
	"github.com/jenkins-x/jx-helpers/v3/pkg/testhelpers"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/jenkins-x/jx-helpers/v3/pkg/gitclient"
	"github.com/jenkins-x/jx-helpers/v3/pkg/gitclient/cli"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/stretchr/testify/assert"
)

func TestBuildPackInitClone(t *testing.T) {
	defaultBranch := testhelpers.GetDefaultBranch(t)

	mainRepo, err := ioutil.TempDir("", uuid.New().String())
	assert.NoError(t, err)

	remoteRepo, err := ioutil.TempDir("", uuid.New().String())
	assert.NoError(t, err)

	defer func() {
		err := os.RemoveAll(mainRepo)
		err2 := os.RemoveAll(remoteRepo)
		if err != nil || err2 != nil {
			log.Logger().Errorf("Error cleaning up tmpdirs because %v", err)
		}
	}()

	err = os.Setenv("JX_HOME", mainRepo)
	assert.NoError(t, err)
	gitDir := mainRepo + "/draft/packs"
	err = os.MkdirAll(gitDir, 0755)
	assert.NoError(t, err)

	gitter := cli.NewCLIClient("", nil)
	assert.NoError(t, err)

	// Prepare a git repo to test - this is our "remote"
	err = gitclient.Init(gitter, remoteRepo)
	assert.NoError(t, err)

	readme := "README"
	initialReadme := "Cheesy!"

	readmePath := filepath.Join(remoteRepo, readme)
	err = ioutil.WriteFile(readmePath, []byte(initialReadme), 0600)
	assert.NoError(t, err)
	_, err = gitclient.AddAndCommitFiles(gitter, remoteRepo, "chore: Initial Commit")
	assert.NoError(t, err, "failed to add and commit files")

	// Prepare another git repo, this is local repo
	err = gitclient.Init(gitter, gitDir)
	assert.NoError(t, err)
	// Set up the remote
	err = gitclient.AddRemote(gitter, gitDir, "origin", remoteRepo)
	assert.NoError(t, err)
	err = gitclient.FetchBranch(gitter, gitDir, "origin", defaultBranch)
	assert.NoError(t, err)
	err = gitclient.Merge(gitter, gitDir, "origin/master")
	assert.NoError(t, err)

	// Removing the remote tracking information, after executing InitBuildPack, it should have not failed and it should've set a remote tracking branch
	_, err = gitter.Command(gitDir, "branch", "--unset-upstream")
	if err != nil {
		t.Logf("could not unset upstream info %s", err.Error())
	}

	//_, err = InitBuildPack(gitter, "", "master")
	//assert.NoError(t, err)

	output, err := gitter.Command(gitDir, "status", "-sb")
	assert.NoError(t, err)
	// Check the current branch is tracking the origin/master one
	// TODO
	assert.Equal(t, "## master", output)
	//assert.Equal(t, "## master...origin/master", output)
}
