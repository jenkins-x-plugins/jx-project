package root_test

import (
	"fmt"
	"testing"

	"github.com/jenkins-x-plugins/jx-project/pkg/cmd/root"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDomainCompare(t *testing.T) {
	testCases := []struct {
		expected  bool
		expectErr bool
		u1        string
		u2        string
	}{
		{
			u1:       "https://github.ibm.com/foo/bar",
			u2:       "https://codeload.github.com/cheese",
			expected: false,
		},
		{
			u1:       "https://github.com/foo/bar",
			u2:       "https://codeload.github.com/cheese",
			expected: true,
		},
		{
			u1:       "https://github.ibm.com/foo/bar",
			u2:       "https://codeload.github.ibm.com/cheese",
			expected: true,
		},
	}

	for _, tc := range testCases {
		message := fmt.Sprintf("comparing %s and %s", tc.u1, tc.u2)
		actual, err := root.SameRootDomain(tc.u1, tc.u2)
		if tc.expectErr {
			require.Error(t, err, "should fail for "+message)
			t.Logf("got expected error for %s of %s\n", message, err.Error())
			continue
		}
		require.NoError(t, err, "failed "+message)
		assert.Equal(t, tc.expected, actual, message)
		t.Logf("%s got %v\n", message, actual)
	}
}
