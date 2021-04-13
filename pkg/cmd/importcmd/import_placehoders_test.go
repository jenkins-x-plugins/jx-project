// +build unit

package importcmd_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/jenkins-x-plugins/jx-project/pkg/cmd/importcmd"
	"github.com/jenkins-x/jx-helpers/v3/pkg/files"

	"github.com/stretchr/testify/assert"
)

func TestReplacePlaceholders(t *testing.T) {
	f, err := ioutil.TempDir("", "test-replace-placeholders")
	assert.NoError(t, err)

	testData := path.Join("test_data", "replace_placeholders")
	_, err = os.Stat(testData)
	assert.NoError(t, err)

	files.CopyDir(testData, f, true)

	assert.NoError(t, err)
	o := importcmd.ImportOptions{}
	//o.Out = testhelpers.Output()
	o.Dir = f
	o.AppName = "bar"
	o.Organisation = "foo"
	o.ScmFactory.NoWriteGitCredentialsFile = true

	o.ReplacePlaceholders("github.com", "registry-org")

	// root file
	testFile, err := LoadBytes(f, "file.txt")
	assert.NoError(t, err)
	assert.Equal(t, "/home/jenkins/go/src/github.com/foo/bar/registry-org", string(testFile), "replaced placeholder")

	// dir1
	testDir1 := path.Join(f, "dir1")
	testFile, err = LoadBytes(testDir1, "file.txt")
	assert.NoError(t, err)
	assert.Equal(t, "/home/jenkins/go/src/github.com/foo/bar/registry-org", string(testFile), "replaced placeholder")

	// dir2
	testDir2 := path.Join(f, "dir2")
	testFile, err = LoadBytes(testDir2, "file.txt")
	assert.NoError(t, err)
	assert.Equal(t, "/home/jenkins/go/src/github.com/foo/bar/registry-org", string(testFile), "replaced placeholder")

	// REPLACE_ME_APP_NAME/REPLACE_ME_APP_NAME.txt
	testDirBar := path.Join(f, "bar")
	testFile, err = LoadBytes(testDirBar, "bar.txt")
	assert.NoError(t, err)
	assert.Equal(t, "/home/jenkins/go/src/github.com/foo/bar/registry-org", string(testFile), "replaced placeholder")

}

// loads a file
func LoadBytes(dir, name string) ([]byte, error) {
	path := filepath.Join(dir, name) // relative path
	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error loading file %s in directory %s, %v", name, dir, err)
	}
	return bytes, nil
}
