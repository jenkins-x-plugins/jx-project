package importcmd_test

import (
	"path/filepath"
	"testing"

	"github.com/jenkins-x/jx-project/pkg/cmd/importcmd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckForUsesImage(t *testing.T) {
	testDir := filepath.Join("test_data", "check_for_uses")

	testCases := []struct {
		path     string
		expected bool
	}{
		{
			path:     "kpt",
			expected: false,
		},
		{
			path:     "uses",
			expected: true,
		},
	}

	for _, tc := range testCases {
		dir := filepath.Join(testDir, tc.path, ".lighthouse", "jenkins-x")
		triggerFile := filepath.Join(dir, "triggers.yaml")
		require.FileExists(t, triggerFile, "file should exist")

		flag, err := importcmd.CheckForUsesImage(dir, triggerFile)
		require.NoError(t, err, "failed to process test %s", tc.path)

		assert.Equal(t, tc.expected, flag, "for test %s", tc.path)
		t.Logf("CheckForUsesImage with %s is %v\n", tc.path, flag)
	}
}
