// +build unit

package quickstarts_test

import (
	"fmt"
	"path/filepath"
	"testing"

	fakejx "github.com/jenkins-x/jx-api/pkg/client/clientset/versioned/fake"
	"github.com/jenkins-x/jx-project/pkg/quickstarts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadQuickStarts(t *testing.T) {
	versionsDir := filepath.Join("test_data", "quickstarts", "version_stream")
	assert.DirExists(t, versionsDir, "no version stream source directory exists")

	ns := "jx"
	o := &quickstarts.Options{
		VersionsDir: versionsDir,
		Namespace:   ns,
		CurrentUser: "",
	}
	o.JXClient = fakejx.NewSimpleClientset()

	model, err := o.LoadQuickStartsModel(nil, false)
	require.NoError(t, err, "LoadQuickStartsModel")

	assert.True(t, len(model.Quickstarts) > 0, "quickstart model should not be empty")

	assertQuickStart(t, model, "node-http", "JavaScript")
	assertQuickStart(t, model, "golang-http", "Go")
}

func assertQuickStart(t *testing.T, model *quickstarts.QuickstartModel, name string, language string) {
	owner := "jenkins-x-quickstarts"
	id := fmt.Sprintf("%s/%s", owner, name)

	qs := model.Quickstarts[id]
	require.NotNil(t, qs, "could not find quickstart for id %s", id)

	assert.Equal(t, owner, qs.Owner, "quickstart.Owner")
	assert.Equal(t, name, qs.Name, "quickstart.Name")
	assert.Equal(t, language, qs.Language, "quickstart.Language")
	assert.Equal(t, id, qs.ID, "quickstart.ID")
}
