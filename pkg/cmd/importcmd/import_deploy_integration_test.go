// +build integration

package importcmd_test

import (
	"fmt"
	"io/ioutil"
	"path"
	"path/filepath"
	"testing"

	v1 "github.com/jenkins-x/jx-api/v4/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx-helpers/v3/pkg/files"
	"github.com/jenkins-x/jx-helpers/v3/pkg/kube/jxenv"
	"github.com/jenkins-x/jx-helpers/v3/pkg/kube/naming"
	"github.com/jenkins-x/jx-project/pkg/cmd/importcmd"
	"github.com/jenkins-x/jx-project/pkg/cmd/testimports"
	"github.com/jenkins-x/jx-project/pkg/constants"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"
)

func TestImportProjectNextGenPipelineWithDeploy(t *testing.T) {
	// TODO
	t.SkipNow()

	t.Parallel()
	tmpDir, err := ioutil.TempDir("", "test-import-deploy-projects-")
	assert.NoError(t, err)
	require.DirExists(t, tmpDir, "could not create temp dir for running tests")

	srcDir := path.Join("test_data", "import_projects", "nodejs")
	assert.DirExists(t, srcDir, "missing source data")

	type testData struct {
		name     string
		callback func(t *testing.T, io *importcmd.ImportOptions, dir string) error
		fail     bool
	}

	tests := []testData{
		{
			name: "team-enable-knative-canary-and-hpa",
			callback: func(t *testing.T, io *importcmd.ImportOptions, dir string) error {
				return assertImportWithDeployTeamSettings(t, io, dir, constants.DeployKindKnative, true, true)
			},
		},
		{
			name: "team-enable-canary-and-hpa",
			callback: func(t *testing.T, io *importcmd.ImportOptions, dir string) error {
				return assertImportWithDeployTeamSettings(t, io, dir, constants.DeployKindDefault, true, true)
			},
		},
		{
			name: "team-enable-canary",
			callback: func(t *testing.T, io *importcmd.ImportOptions, dir string) error {
				return assertImportWithDeployTeamSettings(t, io, dir, constants.DeployKindDefault, true, false)
			},
		},
		{
			name: "team-disable-knative-canary-and-hpa",
			callback: func(t *testing.T, io *importcmd.ImportOptions, dir string) error {
				return assertImportWithDeployTeamSettings(t, io, dir, constants.DeployKindDefault, false, false)
			},
		},
		{
			name: "enable-knative-canary-and-hpa",
			callback: func(t *testing.T, io *importcmd.ImportOptions, dir string) error {
				return assertImportWithDeployCLISettings(t, io, dir, constants.DeployKindKnative, true, true)
			},
		},
		{
			name: "enable-canary-and-hpa",
			callback: func(t *testing.T, io *importcmd.ImportOptions, dir string) error {
				return assertImportWithDeployCLISettings(t, io, dir, constants.DeployKindDefault, true, true)
			},
		},
		{
			name: "enable-canary",
			callback: func(t *testing.T, io *importcmd.ImportOptions, dir string) error {
				return assertImportWithDeployCLISettings(t, io, dir, constants.DeployKindDefault, true, false)
			},
		},
		{
			name: "disable-knative-canary-and-hpa",
			callback: func(t *testing.T, io *importcmd.ImportOptions, dir string) error {
				return assertImportWithDeployCLISettings(t, io, dir, constants.DeployKindDefault, false, false)
			},
		},
	}

	for i, tt := range tests {
		name := tt.name
		if name == "" {
			name = fmt.Sprintf("test%d", i)
		}
		dir := filepath.Join(tmpDir, name)

		err = files.CopyDir(srcDir, dir, true)
		require.NoError(t, err, "failed to copy source to %s", dir)

		_, io := importcmd.NewCmdImportAndOptions()
		io.Dir = dir

		err = tt.callback(t, io, dir)
		if tt.fail {
			require.Error(t, err, "test %s should have failed", name)
			if err != nil {
				t.Logf("test %s got expected error %s", name, err.Error())
			}

		} else {
			require.NoError(t, err, "failed to run test %s", name)
		}

	}
}

func assertImportWithDeployCLISettings(t *testing.T, io *importcmd.ImportOptions, dir string, expectedKind string, expectedCanary bool, expectedHPA bool) error {
	// lets force the CLI arguments to be parsed first to ensure the flags are set to avoid inheriting them from the TeamSettings
	/* TODO
	err := io.Cmd.Flags().Parse(edit.ToDeployArguments("deploy-kind", expectedKind, expectedCanary, expectedHPA))
	if err != nil {
		return err
	}
	*/

	// lets check we parsed the CLI arguments correctly
	_, testName := filepath.Split(dir)
	assert.Equal(t, expectedKind, io.DeployKind, "parse argument: deployKind for test %s", testName)
	assert.Equal(t, expectedCanary, io.DeployOptions.Canary, "parse argument: deployOptions.Canary for test %s", testName)
	assert.Equal(t, expectedHPA, io.DeployOptions.HPA, "parse argument: deployOptions.HPA for test %s", testName)

	io.DeployKind = expectedKind
	io.DeployOptions = v1.DeployOptions{
		Canary: expectedCanary,
		HPA:    expectedHPA,
	}
	return assertImportHasDeploy(t, io, dir, expectedKind, expectedCanary, expectedHPA)
}

func assertImportWithDeployTeamSettings(t *testing.T, io *importcmd.ImportOptions, dir string, expectedKind string, expectedCanary bool, expectedHPA bool) error {
	err := jxenv.ModifyDevEnvironment(io.KubeClient, io.JXClient, io.Namespace, func(env *v1.Environment) error {
		settings := &env.Spec.TeamSettings
		settings.DeployKind = expectedKind
		if !expectedCanary && !expectedHPA {
			settings.DeployOptions = nil
		} else {
			settings.DeployOptions = &v1.DeployOptions{
				Canary: expectedCanary,
				HPA:    expectedHPA,
			}
		}
		return nil
	})
	if err != nil {
		return errors.Wrap(err, "failed to modify team settings")
	}
	return assertImportHasDeploy(t, io, dir, expectedKind, expectedCanary, expectedHPA)
}

func assertImportHasDeploy(t *testing.T, o *importcmd.ImportOptions, testDir string, expectedKind string, expectedCanary bool, expectedHPA bool) error {
	_, testName := filepath.Split(testDir)
	testName = naming.ToValidName(testName)

	testimports.SetFakeClients(t, o)
	o.UseDefaultGit = true

	err := o.Run()
	assert.NoError(t, err, "Failed %s with %s", testName, err)
	if err == nil {
		valuesFile := filepath.Join(testDir, "charts", testName, "values.yaml")
		assert.FileExists(t, filepath.Join(testDir, "charts", testName, "Chart.yaml"))
		assert.FileExists(t, valuesFile)
		t.Logf("completed test in dir %s", testDir)

		// lets validate the resulting values.yaml
		//yamlData, err := ioutil.ReadFile(valuesFile)
		_, err := ioutil.ReadFile(valuesFile)
		assert.NoError(t, err, "Failed to load file %s", valuesFile)

		/* TODO
		eo := edit.EditDeployKindOptions{}
		eo.CommonOptions = o.CommonOptions
		kind, dopts := eo.FindDefaultDeployKindInValuesYaml(string(yamlData))

		assert.Equal(t, expectedKind, kind, "kind for test %s", testName)
		assert.Equal(t, expectedCanary, dopts.Canary, "deployOptions.Canary for test %s", testName)
		assert.Equal(t, expectedHPA, dopts.HPA, "deployOptions.HPA for test %s", testName)
		*/
	}
	return err
}
