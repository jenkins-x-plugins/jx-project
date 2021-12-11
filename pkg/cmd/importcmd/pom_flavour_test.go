// +build unit

package importcmd_test

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/jenkins-x-plugins/jx-project/pkg/cmd/importcmd"
	"github.com/stretchr/testify/assert"
)

func TestMavenIsDefault(t *testing.T) {
	t.Parallel()

	file, err := ioutil.TempFile("", "jx_test")
	assert.Nil(t, err)
	err = ioutil.WriteFile(file.Name(), []byte(""), 0600)
	assert.Nil(t, err)
	flavour, err := importcmd.PomFlavour(file.Name())
	assert.Nil(t, err)
	assert.Equal(t, importcmd.MAVEN, flavour)
	err = os.Remove(file.Name())
	assert.Nil(t, err)
}

func TestMavenJava11Detection(t *testing.T) {
	t.Parallel()

	file, err := ioutil.TempFile("", "jx_test")
	assert.Nil(t, err)
	err = ioutil.WriteFile(file.Name(), []byte("<java.version>11</java.version>"), 0600)
	assert.Nil(t, err)
	flavour, err := importcmd.PomFlavour(file.Name())
	assert.Nil(t, err)
	assert.Equal(t, importcmd.MAVENJAVA11, flavour)
	err = os.Remove(file.Name())
	assert.Nil(t, err)
}

func TestLibertyDetection(t *testing.T) {
	t.Parallel()

	file, err := ioutil.TempFile("", "jx_test")
	assert.Nil(t, err)
	err = ioutil.WriteFile(file.Name(), []byte("<groupId>io.dropwizard"), 0600)
	assert.Nil(t, err)
	flavour, err := importcmd.PomFlavour(file.Name())
	assert.Nil(t, err)
	assert.Equal(t, importcmd.DROPWIZARD, flavour)
	err = os.Remove(file.Name())
	assert.Nil(t, err)
}
