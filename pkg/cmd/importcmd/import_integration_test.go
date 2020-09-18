// +build integration

package importcmd_test

import (
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jenkins-x/jx-helpers/pkg/files"
	"github.com/jenkins-x/jx-helpers/pkg/kube/jxenv"
	"github.com/jenkins-x/jx-helpers/pkg/kube/naming"
	"github.com/jenkins-x/jx-helpers/pkg/testhelpers"
	"github.com/jenkins-x/jx-project/pkg/cmd/importcmd"
	"github.com/jenkins-x/jx-project/pkg/cmd/testimports"
	"github.com/jenkins-x/jx-project/pkg/config"
	"github.com/jenkins-x/jx-project/pkg/constants"

	v1 "github.com/jenkins-x/jx-api/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx-logging/pkg/log"
	"github.com/jenkins-x/jx-project/pkg/jenkinsfile"
	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"
)

const (
	gitSuffix               = "_with_git"
	mavenKeepOldJenkinsfile = "maven_keep_old_jenkinsfile"
	mavenOldJenkinsfile     = "maven_old_jenkinsfile"
	mavenCamel              = "maven_camel"
	mavenSpringBoot         = "maven_springboot"
	probePrefix             = "probePath:"
)

func TestImportProjectsToJenkins(t *testing.T) {
	// TODO jenkins import current disabled
	t.SkipNow()

	tempDir, err := ioutil.TempDir("", "test-import-projects")
	assert.NoError(t, err)

	testData := path.Join("test_data", "import_projects")
	_, err = os.Stat(testData)
	assert.NoError(t, err)

	files, err := ioutil.ReadDir(testData)
	assert.NoError(t, err)

	for _, f := range files {
		if f.IsDir() {
			name := f.Name()
			srcDir := filepath.Join(testData, name)
			testImportProject(t, tempDir, name, srcDir, false, "")
		}
	}
}

func TestImportProjectToJenkinsX(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "test-import-ng-projects")
	assert.NoError(t, err)

	testData := path.Join("test_data", "import_projects")
	_, err = os.Stat(testData)
	assert.NoError(t, err)

	files, err := ioutil.ReadDir(testData)
	assert.NoError(t, err)

	for _, f := range files {
		if f.IsDir() {
			name := f.Name()
			if strings.HasPrefix(name, "maven_keep_old_jenkinsfile") {
				continue
			}
			srcDir := filepath.Join(testData, name)
			testImportProject(t, tempDir, name, srcDir, true, "")
		}
	}
}

func testImportProject(t *testing.T, tempDir string, testcase string, srcDir string, importToJenkinsX bool, buildPackURL string) {
	testDirSuffix := "jenkins"
	if importToJenkinsX {
		testDirSuffix = "jx"
	}
	testDir := filepath.Join(tempDir+"-"+testDirSuffix, testcase)
	files.CopyDir(srcDir, testDir, true)
	if strings.HasSuffix(testcase, gitSuffix) {
		gitDir := filepath.Join(testDir, ".gitdir")
		dotGitExists, gitErr := files.FileExists(gitDir)
		if gitErr != nil {
			log.Logger().Warnf("Git source directory %s does not exist: %s", gitDir, gitErr)
		} else if dotGitExists {
			dotGitDir := filepath.Join(testDir, ".git")
			files.RenameDir(gitDir, dotGitDir, true)
		}
	}
	err := assertImport(t, testDir, testcase, importToJenkinsX, "")
	assert.NoError(t, err, "Importing dir %s from source %s", testDir, srcDir)
}

func assertImport(t *testing.T, testDir string, testcase string, importToJenkinsX bool, buildPackURL string) error {
	_, dirName := filepath.Split(testDir)
	dirName = naming.ToValidName(dirName)
	o := &importcmd.ImportOptions{}

	testimports.SetFakeClients(t, o)
	o.Dir = testDir
	o.DisableMaven = true
	o.UseDefaultGit = true

	if dirName == "maven-camel" {
		o.DeployKind = constants.DeployKindKnative
	}
	if importToJenkinsX {
		o.Destination.JenkinsX.Enabled = true
		callback := func(env *v1.Environment) error {
			env.Spec.TeamSettings.ImportMode = v1.ImportModeTypeYAML
			if buildPackURL != "" {
				env.Spec.TeamSettings.BuildPackURL = buildPackURL
			}
			return nil
		}
		err := jxenv.ModifyDevEnvironment(o.KubeClient, o.JXClient, o.Namespace, callback)
		require.NoError(t, err, "failed to modify Dev Environment")
	} else {
		o.Destination.Jenkins.Enabled = true
		o.Destination.Jenkins.JenkinsName = "myjenkins"
		o.Destination.Jenkins.JenkinsServiceNames = []string{"myjenkins"}

		// lets generate a dummy Jenkinsfile so that we know we don't run the build packs
		jenkinsfile := filepath.Join(testDir, "Jenkinsfile")
		exists, err := files.FileExists(jenkinsfile)
		require.NoError(t, err, "could not check for file %s", jenkinsfile)
		if !exists {
			err = ioutil.WriteFile(jenkinsfile, []byte("node {}"), files.DefaultFileWritePermissions)
			require.NoError(t, err, "failed to write dummy Jenkinsfile to %s", jenkinsfile)
		}
	}

	if testcase == mavenCamel || dirName == mavenSpringBoot {
		o.DisableMaven = testhelpers.TestShouldDisableMaven()
	}

	err := o.Run()
	assert.NoError(t, err, "Failed %s with %s", dirName, err)
	if err == nil {
		defaultJenkinsfileName := jenkinsfile.Name
		defaultJenkinsfileBackupSuffix := jenkinsfile.BackupSuffix
		defaultJenkinsfile := filepath.Join(testDir, defaultJenkinsfileName)
		jfname := defaultJenkinsfile
		if o.Jenkinsfile != "" && o.Jenkinsfile != defaultJenkinsfileName {
			jfname = filepath.Join(testDir, o.Jenkinsfile)
		}
		if dirName == "custom-jenkins" {
			assert.FileExists(t, filepath.Join(testDir, jenkinsfile.Name))
			assert.NoFileExists(t, filepath.Join(testDir, jenkinsfile.Name+".backup"))
			assert.NoFileExists(t, filepath.Join(testDir, jenkinsfile.Name+"-Renamed"))
			if importToJenkinsX {
				assert.FileExists(t, filepath.Join(testDir, config.ProjectConfigFileName))
			} else {
				assert.NoFileExists(t, filepath.Join(testDir, config.ProjectConfigFileName))
			}
		} else if importToJenkinsX {
			assert.NoFileExists(t, jfname)
		} else {
			assert.FileExists(t, jfname)
		}

		if (dirName == "docker" || dirName == "docker-helm") && importToJenkinsX {
			assert.FileExists(t, filepath.Join(testDir, "skaffold.yaml"))
		} else if dirName == "helm" || dirName == "custom-jenkins" || !importToJenkinsX {
			assert.NoFileExists(t, filepath.Join(testDir, "skaffold.yaml"))
		}
		if importToJenkinsX {
			if dirName == "helm" || dirName == "custom-jenkins" {
				assert.NoFileExists(t, filepath.Join(testDir, "Dockerfile"))
			} else {
				assert.FileExists(t, filepath.Join(testDir, "Dockerfile"))
			}
		} else {
			if dirName == "docker" || dirName == "docker-helm" {
				assert.FileExists(t, filepath.Join(testDir, "Dockerfile"))
			} else {
				assert.NoFileExists(t, filepath.Join(testDir, "Dockerfile"))
			}
		}
		if importToJenkinsX {
			if dirName == "docker" || dirName == "custom-jenkins" {
				assert.NoFileExists(t, filepath.Join(testDir, "charts", dirName, "Chart.yaml"))
				assert.NoFileExists(t, filepath.Join(testDir, "charts"))
				if !importToJenkinsX && dirName != "custom-jenkins" {
					testhelpers.AssertFileDoesNotContain(t, jfname, "helm")
				}
			} else {
				assert.FileExists(t, filepath.Join(testDir, "charts", dirName, "Chart.yaml"))
			}
		} else {
			if dirName != "helm" && dirName != "docker-helm" {
				assert.NoFileExists(t, filepath.Join(testDir, "charts", dirName, "Chart.yaml"))
				assert.NoFileExists(t, filepath.Join(testDir, "charts"))
			}
		}

		// lets test we modified the deployment kind
		if dirName == "maven-camel" {
			testhelpers.AssertFileContains(t, filepath.Join(testDir, "charts", "maven-camel", "values.yaml"), "knativeDeploy: true")
		}
		if !importToJenkinsX {
			if strings.HasPrefix(testcase, mavenKeepOldJenkinsfile) {
				testhelpers.AssertFileContains(t, jfname, "THIS IS OLD!")
				assert.NoFileExists(t, jfname+defaultJenkinsfileBackupSuffix)
			} else if strings.HasPrefix(testcase, mavenOldJenkinsfile) {
				assert.FileExists(t, jfname)
			}
		}

		if !o.DisableMaven {
			if testcase == mavenCamel {
				// should have modified it
				assertProbePathEquals(t, filepath.Join(testDir, "charts", dirName, "values.yaml"), "/health")
			}
			if testcase == mavenSpringBoot {
				// should have left it
				assertProbePathEquals(t, filepath.Join(testDir, "charts", dirName, "values.yaml"), "/actuator/health")
			}
		}
	}
	return err
}

func assertProbePathEquals(t *testing.T, fileName string, expectedProbe string) {
	if assert.FileExists(t, fileName) {
		data, err := ioutil.ReadFile(fileName)
		assert.NoError(t, err, "Failed to read file %s", fileName)
		if err == nil {
			text := string(data)
			found := false
			lines := strings.Split(text, "\n")

			for _, line := range lines {
				if strings.HasPrefix(line, probePrefix) {
					found = true
					value := strings.TrimSpace(strings.TrimPrefix(line, probePrefix))
					assert.Equal(t, expectedProbe, value, "file %s probe with key: %s", fileName, probePrefix)
					break
				}

			}
			assert.True(t, found, "No probe found in file %s with key: %s", fileName, probePrefix)
		}
	}
}
