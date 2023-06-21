//go:build unit
// +build unit

package importcmd_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jenkins-x-plugins/jx-project/pkg/cmd/importcmd"
	"github.com/stretchr/testify/assert"
)

func TestMavenIsDefault(t *testing.T) {
	t.Parallel()

	genericMaven(t, "", importcmd.MAVEN, "")
}

func TestMavenJava17Detection(t *testing.T) {
	t.Parallel()

	pack := "maven-java17"
	packDir, err := os.MkdirTemp("", "jx_test")
	assert.Nil(t, err)
	os.Mkdir(filepath.Join(packDir, pack), 0700)
	assert.Nil(t, err)
	genericMaven(t, packDir, pack, "<maven.compiler.release>17</maven.compiler.release>")
	err = os.RemoveAll(packDir)
	assert.Nil(t, err)
}

func TestMavenJava11Detection(t *testing.T) {
	t.Parallel()

	pack := "maven-java11"
	packDir, err := os.MkdirTemp("", "jx_test")
	assert.Nil(t, err)
	os.Mkdir(filepath.Join(packDir, pack), 0700)
	assert.Nil(t, err)
	genericMaven(t, packDir, pack, "<java.version>11</java.version>")
	err = os.RemoveAll(packDir)
	assert.Nil(t, err)
}

func TestMavenDefaultJavaDetection(t *testing.T) {
	t.Parallel()

	genericMaven(t, ".", importcmd.MAVEN, "<java.version>11</java.version>")
}

func TestLibertyDetection(t *testing.T) {
	t.Parallel()

	genericMaven(t, "", importcmd.DROPWIZARD, "<groupId>io.dropwizard")
}

func genericMaven(t *testing.T, packsDir, expectedFlavour, testFileContent string) {
	file, err := os.CreateTemp("", "jx_test")
	assert.Nil(t, err)
	err = os.WriteFile(file.Name(), []byte(testFileContent), 0600)
	assert.Nil(t, err)
	flavour, err := importcmd.PomFlavour(packsDir, file.Name())
	assert.Nil(t, err)
	assert.Equal(t, expectedFlavour, flavour)
	err = os.Remove(file.Name())
	assert.Nil(t, err)
}
