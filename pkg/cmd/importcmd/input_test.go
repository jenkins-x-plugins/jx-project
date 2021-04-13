package importcmd_test

import (
	"context"
	"testing"

	"github.com/jenkins-x-plugins/jx-project/pkg/cmd/importcmd"
	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/jx-helpers/v3/pkg/scmhelpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockRepositoryService struct {
	mock.Mock
	scm.RepositoryService
}

func (m *MockRepositoryService) Find(ctx context.Context, repo string) (*scm.Repository, *scm.Response, error) {
	args := m.Called(ctx, repo)
	return args.Get(0).(*scm.Repository), args.Get(1).(*scm.Response), args.Error(2)
}

func TestShouldReturnErrorWhenNoErrorReturnedIndicatingSuccessResponseAndRepositoryExists(t *testing.T) {
	t.Parallel()
	m := new(MockRepositoryService)
	c := scm.Client{
		Repositories: m,
	}
	cmd := &importcmd.ImportOptions{
		ScmFactory: scmhelpers.Factory{
			ScmClient: &c,
		},
	}
	cmd.ScmFactory.NoWriteGitCredentialsFile = true
	o := "proj"
	r := "test"
	res := new(scm.Response)
	rep := new(scm.Repository)
	m.On("Find", mock.Anything, o+"/"+r).Return(rep, res, nil)

	result := cmd.ValidateRepositoryName(o, r)

	assert.Error(t, result, "Should return an error")
}

func TestShouldReturnErrorWhenErrorReturnedAndResponseIsSuccessful(t *testing.T) {
	t.Parallel()
	m := new(MockRepositoryService)
	c := scm.Client{
		Repositories: m,
	}
	cmd := &importcmd.ImportOptions{
		ScmFactory: scmhelpers.Factory{
			ScmClient: &c,
		},
	}
	cmd.ScmFactory.NoWriteGitCredentialsFile = true
	o := "proj"
	r := "test"
	res := new(scm.Response)
	res.Status = 200
	rep := new(scm.Repository)
	err := scm.ErrNotAuthorized
	m.On("Find", mock.Anything, o+"/"+r).Return(rep, res, err)

	result := cmd.ValidateRepositoryName(o, r)

	assert.Error(t, result, "Should return an error")
}

func TestShouldReturnNilWhenErrorReturnedAndResponseIsNotFound(t *testing.T) {
	t.Parallel()
	m := new(MockRepositoryService)
	c := scm.Client{
		Repositories: m,
	}
	cmd := &importcmd.ImportOptions{
		ScmFactory: scmhelpers.Factory{
			ScmClient: &c,
		},
	}
	cmd.ScmFactory.NoWriteGitCredentialsFile = true
	o := "proj"
	r := "test"
	res := new(scm.Response)
	res.Status = 404
	rep := new(scm.Repository)
	err := scm.ErrNotFound
	m.On("Find", mock.Anything, o+"/"+r).Return(rep, res, err)

	result := cmd.ValidateRepositoryName(o, r)

	assert.Nil(t, result, "Should return nil value")
}
