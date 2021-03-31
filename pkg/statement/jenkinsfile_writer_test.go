// +build unit

package statement_test

import (
	"testing"

	"github.com/jenkins-x-plugins/jx-project/pkg/statement"
	"github.com/stretchr/testify/assert"
)

func TestJenkinsfileWriter(t *testing.T) {
	expectedValue := `container('maven') {
  dir('/foo/bar') {
    sh "ls -al"
    sh "mvn deploy"
  }
}
`
	writer := statement.NewWriter(0)

	statements := []*statement.Statement{
		{
			Function:  "container",
			Arguments: []string{"maven"},
			Children: []*statement.Statement{
				{
					Function:  "dir",
					Arguments: []string{"/foo/bar"},
					Children: []*statement.Statement{
						{
							Statement: "sh \"ls -al\"",
						},
					},
				},
			},
		},
		{
			Function:  "container",
			Arguments: []string{"maven"},
			Children: []*statement.Statement{
				{
					Function:  "dir",
					Arguments: []string{"/foo/bar"},
					Children: []*statement.Statement{
						{
							Statement: "sh \"mvn deploy\"",
						},
					},
				},
			},
		},
	}
	writer.Write(statements)
	text := writer.String()

	assert.Equal(t, expectedValue, text, "for statements %#v", statements)
}
