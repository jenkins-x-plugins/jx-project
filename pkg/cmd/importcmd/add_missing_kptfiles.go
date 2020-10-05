package importcmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/jenkins-x/jx-helpers/pkg/files"
	"github.com/jenkins-x/jx-helpers/pkg/gitclient"
	"github.com/jenkins-x/jx-helpers/pkg/gitclient/gitdiscovery"
	"github.com/jenkins-x/jx-helpers/pkg/stringhelpers"
	"github.com/jenkins-x/jx-logging/pkg/log"
	"github.com/pkg/errors"
)

var (
	kptFile = `apiVersion: kpt.dev/v1alpha1
kind: Kptfile
metadata:
    name: %s
upstream:
    type: git
    git:
        commit: %s
        repo: %s
        directory: %s
        ref: master
`
)

// createMissingLighthouseKptFiles lets create any missing Kptfile for any .lighthouse/somedir directories
// so that after the pipeline folder has been added we can later on upgrade it from its source via kpt
func (o *ImportOptions) createMissingLighthouseKptFiles(lighthouseDir, packName string) error {
	fileSlice, err := ioutil.ReadDir(lighthouseDir)
	if err != nil {
		return errors.Wrapf(err, "failed to read dir %s", lighthouseDir)
	}
	for _, f := range fileSlice {
		if !f.IsDir() {
			continue
		}
		name := f.Name()

		triggerFile := filepath.Join(lighthouseDir, name, "triggers.yaml")
		exists, err := files.FileExists(triggerFile)
		if err != nil {
			return errors.Wrapf(err, "failed to check if file exists %s", triggerFile)
		}

		if !exists {
			continue
		}

		// lets check if we have a local Kptfile for this trigger folder
		localKptDir := filepath.Join(o.Dir, ".lighthouse", name)
		localKptFile := filepath.Join(localKptDir, "Kptfile")
		exists, err = files.FileExists(localKptFile)
		if err != nil {
			return errors.Wrapf(err, "failed to check if file exists %s", localKptFile)
		}
		if !exists {
			sha, err := gitclient.GetLatestCommitSha(o.Git(), lighthouseDir)
			if err != nil {
				return errors.Wrapf(err, "failed to discover latest git commit for dir %s", lighthouseDir)
			}

			gitURL, err := gitdiscovery.FindGitURLFromDir(lighthouseDir)
			if err != nil {
				return errors.Wrapf(err, "failed to discover git URL in dir %s", lighthouseDir)
			}

			// lets remove any user/passwords just in case
			gitURL = stringhelpers.SanitizeURL(gitURL)

			if gitURL == "" {
				return errors.Errorf("failed to find git URL in dir %s", lighthouseDir)
			}

			err = os.MkdirAll(localKptDir, files.DefaultDirWritePermissions)
			if err != nil {
				return errors.Wrapf(err, "failed to create dir %s", localKptDir)
			}

			fromDir := filepath.Join("/packs", packName, ".lighthouse", name)
			gitURL = strings.TrimSuffix(gitURL, ".git")
			text := fmt.Sprintf(kptFile, name, sha, gitURL, fromDir)
			err = ioutil.WriteFile(localKptFile, []byte(text), files.DefaultFileWritePermissions)
			if err != nil {
				return errors.Wrapf(err, "failed to save file %s", localKptFile)
			}

			log.Logger().Infof("created %s", localKptFile)
		}
	}
	return nil
}
